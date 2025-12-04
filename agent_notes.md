
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


