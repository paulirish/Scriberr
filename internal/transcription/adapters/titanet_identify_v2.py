#!/usr/bin/env python3
"""
TitaNet Speaker Identification Script (v2 with Confidence-Weighted EMA and Imposter Collection)
"""
import argparse
import json
import os
import sys
import torch
import numpy as np
from typing import List, Dict
import logging
import uuid

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

try:
    from nemo.collections.asr.models import EncDecSpeakerLabelModel
    from qdrant_client import QdrantClient
    from qdrant_client.http import models as qmodels
except ImportError as e:
    logger.error(f"Import Error: {e}")
    sys.exit(1)

def setup_qdrant(host: str, collection_name: str, vector_size: int = 192):
    client = QdrantClient(host=host, port=6333)
    collections = client.get_collections().collections
    exists = any(c.name == collection_name for c in collections)
    if not exists:
        logger.info(f"Creating collection {collection_name}")
        client.create_collection(
            collection_name=collection_name,
            vectors_config=qmodels.VectorParams(size=vector_size, distance=qmodels.Distance.COSINE),
        )
    return client

def identify_speakers(
    audio_path: str,
    segments_file: str,
    output_file: str,
    qdrant_host: str = "qdrant",
    collection_name: str = "speakers",
    threshold: float = 0.7,
    threshold_new: float = 0.55,
    alpha_max: float = 0.25,
    min_duration_for_full_weight: float = 4.0,
    device: str = "auto"
):
    # 1. Load Model
    if device == "auto":
        device = "cuda" if torch.cuda.is_available() else "cpu"
    logger.info(f"Loading TitaNet on {device}")
    script_dir = os.path.dirname(os.path.abspath(__file__))
    model_path = os.path.join(script_dir, "titanet-l.nemo")
    model = EncDecSpeakerLabelModel.restore_from(restore_path=model_path, map_location=device)
    model.eval()

    # 2. Setup Qdrant
    client = setup_qdrant(qdrant_host, collection_name, vector_size=192)
    imposter_collection_name = "imposter_candidates"
    setup_qdrant(qdrant_host, imposter_collection_.name, vector_size=192)

    # 3. Load Segments
    with open(segments_file, 'r') as f:
        data = json.load(f)
        segments = data.get("segments", [])

    local_speakers = {}
    for idx, seg in enumerate(segments):
        spk = seg.get("speaker")
        if spk not in local_speakers:
            local_speakers[spk] = []
        local_speakers[spk].append(idx)

    # 4. Process each local speaker
    global_mapping = {}
    import soundfile as sf
    full_waveform, sample_rate = sf.read(audio_path)
    full_waveform = torch.from_numpy(full_waveform).float()
    if full_waveform.ndim > 1: full_waveform = full_waveform.mean(dim=0)
    full_waveform = full_waveform.unsqueeze(0).to(device)

    for local_spk, indices in local_speakers.items():
        logger.info(f"Processing local speaker {local_spk} ({len(indices)} segments)")
        embeddings = []
        indices.sort(key=lambda i: segments[i].get("end") - segments[i].get("start"), reverse=True)
        top_indices = indices[:10]
        
        total_duration = 0
        for idx in top_indices:
            seg = segments[idx]
            start, end = seg["start"], seg["end"]
            duration = end - start
            total_duration += duration

            start_frame, end_frame = int(start * sample_rate), int(end * sample_rate)
            if end_frame > full_waveform.shape[1]: end_frame = full_waveform.shape[1]
            sub_audio = full_waveform[:, start_frame:end_frame]
            
            if sub_audio.shape[1] < 1600: continue
            len_tensor = torch.tensor([sub_audio.shape[1]], device=device)
            with torch.no_grad():
                _, embs = model(input_signal=sub_audio, input_signal_length=len_tensor)
                embeddings.append(embs[0].cpu().numpy())

        if not embeddings:
            logger.warning(f"No valid embeddings for {local_spk}")
            continue

        centroid = np.mean(embeddings, axis=0)
        norm = np.linalg.norm(centroid)
        if norm > 0: centroid = centroid / norm

        # 5. Query Qdrant
        search_result = client.search(
            collection_name=collection_name,
            query_vector=centroid.tolist(),
            limit=1,
            score_threshold=threshold_new, # Use lower bound for initial query
            with_vector=True
        )

        best_match_score = search_result[0].score if search_result else 0.0
        
        if best_match_score >= threshold:
            # High confidence match
            best_match = search_result[0]
            global_id = best_match.payload.get("name", "Unknown")
            logger.info(f"Matched {local_spk} to {global_id} (score: {best_match.score:.4f})")

            existing_centroid = np.array(best_match.vector)
            wt = min(alpha_max, total_duration / min_duration_for_full_weight)
            
            logger.info(f"Updating speaker {global_id} with weight {wt:.4f} from {total_duration:.2f}s of audio")
            updated_centroid = ((1 - wt) * existing_centroid) + (wt * centroid)
            
            norm = np.linalg.norm(updated_centroid)
            if norm > 0: updated_centroid = updated_centroid / norm
            
            client.upsert(
                collection_name=collection_name,
                points=[qmodels.PointStruct(id=best_match.id, vector=updated_centroid.tolist(), payload=best_match.payload)]
            )
            global_mapping[local_spk] = global_id

        elif best_match_score < threshold_new:
            # Confirmed non-match -> Enroll new speaker and collect imposter embedding
            new_id = str(uuid.uuid4())
            human_name = f"Speaker-{new_id[:8]}"
            logger.info(f"Enrolling {local_spk} as new speaker {human_name} (score: {best_match_score:.4f} < {threshold_new})")
            client.upsert(
                collection_name=collection_name,
                points=[qmodels.PointStruct(id=new_id, vector=centroid.tolist(), payload={"name": human_name, "created_at": str(os.path.getctime(audio_path))})]
            )
            global_mapping[local_spk] = human_name

            # Also add to imposter candidates
            client.upsert(
                collection_name=imposter_collection_name,
                points=[qmodels.PointStruct(id=str(uuid.uuid4()), vector=centroid.tolist())]
            )
        else:
            # Ambiguity zone: treat as a new speaker for now, but don't add to imposters
            logger.info(f"Ambiguous match for {local_spk} (score: {best_match_score:.4f}). Enrolling as new temporary speaker.")
            new_id = str(uuid.uuid4())
            human_name = f"Speaker-{new_id[:8]}"
            client.upsert(
                collection_name=collection_name,
                points=[qmodels.PointStruct(id=new_id, vector=centroid.tolist(), payload={"name": human_name, "created_at": str(os.path.getctime(audio_path))})]
            )
            global_mapping[local_spk] = human_name

    # 6. Update Segments and Save
    for seg in segments:
        local = seg.get("speaker")
        if local in global_mapping:
            seg["speaker"] = global_mapping[local]
    
    output_data = {"segments": segments}
    with open(output_file, 'w') as f:
        json.dump(output_data, f, indent=2)

    logger.info(f"Identification complete. Saved to {output_file}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="TitaNet Speaker Identification")
    parser.add_argument("audio_file")
    parser.add_argument("segments_file")
    parser.add_argument("output_file")
    parser.add_argument("--qdrant", default="qdrant")
    parser.add_argument("--collection", default="speakers")
    parser.add_argument("--threshold", type=float, default=0.7, help="Threshold for matching an existing speaker.")
    parser.add_argument("--threshold-new", type=float, default=0.55, help="Threshold below which a new speaker is definitely enrolled.")
    parser.add_argument("--alpha-max", type=float, default=0.25, help="Max learning rate for EMA.")
    parser.add_argument("--min-duration-full-weight", type=float, default=4.0, help="Min duration for full weight in EMA.")
    args = parser.parse_args()

    identify_speakers(
        args.audio_file,
        args.segments_file,
        args.output_file,
        qdrant_host=args.qdrant,
        collection_name=args.collection,
        threshold=args.threshold,
        threshold_new=args.threshold_new,
        alpha_max=args.alpha_max,
        min_duration_for_full_weight=args.min_duration_for_full_weight,
    )
