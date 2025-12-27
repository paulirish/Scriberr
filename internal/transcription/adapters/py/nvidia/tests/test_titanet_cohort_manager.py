import unittest
from unittest.mock import MagicMock, patch, call
import sys
import os
import numpy as np

# Add the adapters directory to the Python path
sys.path.insert(
    0,
    os.path.abspath(
        os.path.join(os.path.dirname(__file__), "../../internal/transcription/adapters")
    ),
)

# Now import the script we want to test
import titanet_cohort_manager as cohort_manager


class TestCohortManager(unittest.TestCase):
    @patch("titanet_cohort_manager.QdrantClient")
    def test_refresh_cohort_success(self, MockQdrantClient):
        """
        Test the successful refresh of the S-Norm cohort from a list of imposters.
        """
        # --- Setup Mocks ---
        mock_client_instance = MockQdrantClient.return_value

        # Mock collection info
        mock_collection_info = MagicMock()
        mock_collection_info.vectors_count = 10
        mock_client_instance.get_collection.return_value = mock_collection_info

        # Mock imposter vectors
        mock_imposter_points = [MagicMock(vector=[float(i)] * 192) for i in range(10)]
        mock_client_instance.scroll.return_value = (mock_imposter_points, None)

        # --- Run Function ---
        cohort_manager.refresh_cohort(
            qdrant_host="mock_host",
            imposter_collection="imposters",
            cohort_collection="cohort",
            sample_size=5,
        )

        # --- Assertions ---
        # 1. Check if it fetched collection info
        mock_client_instance.get_collection.assert_called_once_with(
            collection_name="imposters"
        )

        # 2. Check if it scrolled through imposters
        mock_client_instance.scroll.assert_called_once_with(
            collection_name="imposters", limit=10, with_payload=False, with_vectors=True
        )

        # 3. Check if it recreated the cohort collection
        mock_client_instance.recreate_collection.assert_called_once()
        args, kwargs = mock_client_instance.recreate_collection.call_args
        self.assertEqual(kwargs.get("collection_name"), "cohort")

        # 4. Check if it upserted the new sampled points
        mock_client_instance.upsert.assert_called_once()
        args, kwargs = mock_client_instance.upsert.call_args
        self.assertEqual(kwargs.get("collection_name"), "cohort")
        self.assertEqual(
            len(kwargs.get("points")), 5
        )  # Check if the sample size was respected

    @patch("titanet_cohort_manager.QdrantClient")
    def test_refresh_cohort_no_imposters(self, MockQdrantClient):
        """
        Test that the refresh process handles the case where there are no imposters.
        """
        # --- Setup Mocks ---
        mock_client_instance = MockQdrantClient.return_value
        mock_collection_info = MagicMock()
        mock_collection_info.vectors_count = 0  # No imposters
        mock_client_instance.get_collection.return_value = mock_collection_info

        # --- Run Function ---
        cohort_manager.refresh_cohort(
            qdrant_host="mock_host",
            imposter_collection="imposters",
            cohort_collection="cohort",
            sample_size=5,
        )

        # --- Assertions ---
        # 1. Check if it fetched collection info
        mock_client_instance.get_collection.assert_called_once_with(
            collection_name="imposters"
        )

        # 2. Ensure it did NOT proceed to scroll, recreate, or upsert
        mock_client_instance.scroll.assert_not_called()
        mock_client_instance.recreate_collection.assert_not_called()
        mock_client_instance.upsert.assert_not_called()

    @patch("titanet_cohort_manager.QdrantClient")
    def test_sampling_less_than_sample_size(self, MockQdrantClient):
        """
        Test that sampling works correctly when available imposters are less than sample_size.
        """
        # --- Setup Mocks ---
        mock_client_instance = MockQdrantClient.return_value
        mock_collection_info = MagicMock()
        mock_collection_info.vectors_count = 3  # Only 3 imposters available
        mock_client_instance.get_collection.return_value = mock_collection_info
        mock_imposter_points = [MagicMock(vector=[1.0] * 192)] * 3
        mock_client_instance.scroll.return_value = (mock_imposter_points, None)

        # --- Run Function ---
        cohort_manager.refresh_cohort(
            qdrant_host="mock_host",
            imposter_collection="imposters",
            cohort_collection="cohort",
            sample_size=10,  # Request more than available
        )

        # --- Assertions ---
        # Check that it upserted only the number of points available (3)
        mock_client_instance.upsert.assert_called_once()
        args, kwargs = mock_client_instance.upsert.call_args
        self.assertEqual(len(kwargs.get("points")), 3)


if __name__ == "__main__":
    unittest.main()
