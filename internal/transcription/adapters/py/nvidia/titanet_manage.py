#!/usr/bin/env python3
"""
TitaNet Speaker Management Script
"""
import argparse
import json
import sys
import logging
from typing import List, Dict

# Setup logging
logging.basicConfig(level=logging.ERROR, format='%(message)s')
logger = logging.getLogger(__name__)

try:
    from qdrant_client import QdrantClient
    from qdrant_client.http import models as qmodels
except ImportError as e:
    logger.error(f"Import Error: {e}")
    sys.exit(1)

def setup_client(host: str):
    return QdrantClient(host=host, port=6333)

def list_speakers(host: str, collection: str):
    client = setup_client(host)
    try:
        # Scroll through points to get all speakers
        # Note: This might be slow for huge collections, but fine for <10k speakers
        points, _ = client.scroll(
            collection_name=collection,
            limit=1000,
            with_payload=True,
            with_vectors=False
        )

        speakers = []
        for p in points:
            payload = p.payload or {}
            speakers.append({
                "id": p.id,
                "name": payload.get("name", "Unknown"),
                "created_at": float(payload.get("created_at", 0))
            })

        print(json.dumps(speakers))
    except Exception as e:
        logger.error(f"Error listing speakers: {e}")
        sys.exit(1)

func get_speaker(host: str, collection: str, speaker_id: str):
    client = setup_client(host)
    try:
        points = client.retrieve(
            collection_name=collection,
            ids=[speaker_id],
            with_payload=True,
            with_vectors=False
        )
        if not points:
            logger.error("Speaker not found")
            sys.exit(1)

        p = points[0]
        payload = p.payload or {}
        speaker = {
            "id": p.id,
            "name": payload.get("name", "Unknown"),
            "created_at": float(payload.get("created_at", 0))
        }
        print(json.dumps(speaker))
    except Exception as e:
        logger.error(f"Error getting speaker: {e}")
        sys.exit(1)

def rename_speaker(host: str, collection: str, speaker_id: str, new_name: str):
    client = setup_client(host)
    try:
        # Verify existence
        points = client.retrieve(
            collection_name=collection,
            ids=[speaker_id]
        )

        if not points:
            logger.error("Speaker not found")
            sys.exit(1)

        # Update payload
        client.set_payload(
            collection_name=collection,
            payload={"name": new_name},
            points=[speaker_id]
        )
        print(json.dumps({"status": "success", "id": speaker_id, "name": new_name}))
    except Exception as e:
        logger.error(f"Error renaming speaker: {e}")
        sys.exit(1)

def delete_speaker(host: str, collection: str, speaker_id: str):
    client = setup_client(host)
    try:
        client.delete(
            collection_name=collection,
            points_selector=qmodels.PointIdsList(points=[speaker_id])
        )
        print(json.dumps({"status": "success", "id": speaker_id}))
    except Exception as e:
        logger.error(f"Error deleting speaker: {e}")
        sys.exit(1)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)

    # List
    parser_list = subparsers.add_parser("list")
    parser_list.add_argument("--qdrant", default="qdrant")
    parser_list.add_argument("--collection", default="speakers")

    # Get
    parser_get = subparsers.add_parser("get")
    parser_get.add_argument("id")
    parser_get.add_argument("--qdrant", default="qdrant")
    parser_get.add_argument("--collection", default="speakers")

    # Rename
    parser_rename = subparsers.add_parser("rename")
    parser_rename.add_argument("id")
    parser_rename.add_argument("new_name")
    parser_rename.add_argument("--qdrant", default="qdrant")
    parser_rename.add_argument("--collection", default="speakers")

    # Delete
    parser_delete = subparsers.add_parser("delete")
    parser_delete.add_argument("id")
    parser_delete.add_argument("--qdrant", default="qdrant")
    parser_delete.add_argument("--collection", default="speakers")

    args = parser.parse_args()

    if args.command == "list":
        list_speakers(args.qdrant, args.collection)
    elif args.command == "get":
        get_speaker(args.qdrant, args.collection, args.id)
    elif args.command == "rename":
        rename_speaker(args.qdrant, args.collection, args.id, args.new_name)
    elif args.command == "delete":
        delete_speaker(args.qdrant, args.collection, args.id)
