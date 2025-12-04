
## Implementing Speaker System v2

I will now begin implementing the `speaker_system_v2.md` design. I will break this down into two main parts:

1.  **Confidence-Weighted EMA:** Implement the adaptive learning rate for centroid updates.
2.  **Self-Tuning Adaptive S-Norm:** Implement the dynamic cohort for score normalization.

### Part 1: Confidence-Weighted EMA

**Current state:** The `titanet_identify.py` script does not update existing speaker profiles. It only enrolls new speakers.

**Plan:**
1.  Modify `titanet_identify.py` to fetch the existing centroid when a speaker is matched.
2.  Implement the Confidence-Weighted EMA logic to calculate the new centroid.
3.  Update the centroid in Qdrant.

I will start with this part.



### Part 2: Self-Tuning Adaptive S-Norm

**Current state:** The system uses a fixed similarity threshold.

**Plan:**

1.  **Imposter Candidate Collection:**
    *   In `titanet_identify.py`, when a segment is a clear non-match (e.g., score below a new, lower threshold `τ_new`), I will store its embedding in a new Qdrant collection, e.g., `imposter_candidates`.
2.  **Cohort Management Script:**
    *   Create a new Python script, `titanet_cohort_manager.py`.
    *   This script will have functions to:
        *   Create the `imposter_candidates` collection if it doesn't exist.
        *   Create a `snorm_cohort` collection.
        *   Run a background job to sample from `imposter_candidates` and refresh `snorm_cohort`.
    *   I will add a new function `EnsureCohortManagementScript` to `titanet_adapter.go` to write this script.
3.  **S-Norm in Identification:**
    *   In `titanet_identify.py`, before making a decision, I will:
        *   Fetch the current `snorm_cohort`.
        *   Calculate the S-Norm score for the input embedding against the candidate centroid.
        *   Use the normalized score against the threshold.

This is a significant change, so I will implement it step by step. I'll start with step 1.


