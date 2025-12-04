#!/usr/bin/env python3
"""
TitaNet S-Norm Cohort Management Script
"""
import argparse
import logging
import random
import sys
import uuid # Added import

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

try:
    from qdrant_client import QdrantClient
    from qdrant_client.http import models as qmodels
except ImportError as e:
    logger.error(f"Import Error: {e}. Please ensure qdrant-client is installed.")
    sys.exit(1)

def setup_qdrant(host: str, collection_name: str, vector_size: int = 192):
    """Ensure a Qdrant collection exists."""
    client = QdrantClient(host=host, port=6333)
    collections = client.get_collections().collections
    exists = any(c.name == collection_name for c in collections)
    if not exists:
        logger.info(f"Creating collection '{collection_name}'")
        client.create_collection(
            collection_name=collection_name,
            vectors_config=qmodels.VectorParams(size=vector_size, distance=qmodels.Distance.COSINE),
        )
    return client

def refresh_cohort(
    qdrant_host: str,
    imposter_collection: str,
    cohort_collection: str,
    sample_size: int
):
    """
    Refreshes the S-Norm cohort by sampling from the imposter candidates.
    """
    client = setup_qdrant(qdrant_host, cohort_collection)
    
    logger.info(f"Attempting to refresh cohort '{cohort_collection}' from '{imposter_collection}'")

    try:
        # 1. Get total number of imposter candidates
        imposter_collection_info = client.get_collection(collection_name=imposter_collection)
        total_imposters = imposter_collection_info.vectors_count
        
        logger.info(f"Found {total_imposters} total imposter candidates.")

        if total_imposters == 0:
            logger.warning("No imposter candidates available to build a cohort. Exiting.")
            return

        # 2. Fetch all imposter vectors
        # Using scroll to get all points. This is okay for a background job.
        all_imposters, _ = client.scroll(
            collection_name=imposter_collection,
            limit=total_imposters,
            with_payload=False,  # We only need the vectors
            with_vectors=True
        )

        # 3. Sample from the retrieved points
        num_to_sample = min(sample_size, len(all_imposters))
        logger.info(f"Sampling {num_to_sample} vectors for the new cohort.")
        sampled_points = random.sample(all_imposters, num_to_sample)

        # 4. Recreate the cohort collection with the new samples
        # Recreating ensures the cohort only contains the new sample.
        client.recreate_collection(
            collection_name=cohort_collection,
            vectors_config=qmodels.VectorParams(size=192, distance=qmodels.Distance.COSINE),
        )

        # Prepare points for upsertion with new UUIDs for the cohort collection
        new_cohort_points = [
            qmodels.PointStruct(
                id=str(uuid.uuid4()),
                vector=point.vector
            ) for point in sampled_points
        ]

        client.upsert(
            collection_name=cohort_collection,
            points=new_cohort_points,
            wait=True
        )

        logger.info(f"Successfully refreshed cohort '{cohort_collection}' with {len(new_cohort_points)} vectors.")

    except Exception as e:
        logger.error(f"An error occurred during cohort refresh: {e}")
        # Depending on the error, you might want to add more specific handling.
        # For instance, if the imposter collection doesn't exist, we should log that clearly.
        if "not found" in str(e).lower():
            logger.error(f"Collection '{imposter_collection}' not found. Please ensure it exists and has data.")
        sys.exit(1)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="TitaNet S-Norm Cohort Management")
    parser.add_argument("--qdrant", default="qdrant", help="Qdrant host.")
    parser.add_argument(
        "--imposter-collection",
        default="imposter_candidates",
        help="Name of the collection containing imposter embeddings."
    )
    parser.add_argument(
        "--cohort-collection",
        default="snorm_cohort",
        help="Name of the collection to store the S-Norm cohort."
    )
    parser.add_argument(
        "--sample-size",
        type=int,
        default=500,
        help="Number of imposter embeddings to sample for the cohort."
    )

    args = parser.parse_args()

    # Import uuid here as it's only needed for the main execution block
    import uuid

    refresh_cohort(
        qdrant_host=args.qdrant,
        imposter_collection=args.imposter_collection,
        cohort_collection=args.cohort_collection,
        sample_size=args.sample_size
    )
