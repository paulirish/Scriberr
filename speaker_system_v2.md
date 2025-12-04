# Speaker Identification System v2: Advanced Robustness Mechanisms

This document outlines architectural enhancements to the speaker identification system, incorporating feedback to improve robustness against real-world acoustic variability and uncertainty. The goal is to evolve from a baseline implementation to a production-grade system that is more accurate, stable, and adaptive.

The core proposals address two critical areas:
1.  **Aleatoric Uncertainty in Profile Updates:** Preventing low-quality or short audio segments from disproportionately corrupting stable speaker profiles.
2.  **Environmental Acoustic Drift:** Ensuring the scoring and decision-making logic adapts to the specific operational environment rather than relying on generic, mismatched assumptions.

---

## **1. Advanced Centroid Management: Confidence-Weighted EMA for Aleatoric Uncertainty**

The stability of a speaker's profile (centroid) is paramount. The previous design employed an Exponential Moving Average (EMA) with a fixed learning rate (`α`), which treats all incoming audio segments equally. This is a significant vulnerability.

### **1.1. The Motivation: Resisting "Noise Poisoning"**

Real-world audio is imperfect. A speaker's profile can be "poisoned" by short, noisy, or distorted segments, causing the centroid to drift away from its true position. A 5-second, high-Signal-to-Noise-Ratio (SNR) segment should have a much stronger influence on the centroid than a 1-second segment corrupted by background noise. This is a direct mitigation strategy for **Aleatoric Uncertainty**—the inherent randomness and unpredictability in the data itself.

### **1.2. The v2 Implementation: Adaptive Learning Rate via Quality Weighting**

We will replace the fixed learning rate `α` with an adaptive, per-segment weight, `Wt`, that is directly proportional to the quality and duration of the audio segment.

The EMA update rule evolves from `μ_k,t = (1 - α) * μ_k,t-1 + α * e_t` to:

**`μ_k,t = (1 - Wt) * μ_k,t-1 + Wt * e_t`**

The adaptive weight `Wt` will be calculated as:

**`Wt = min(α_max, segment_duration_seconds / min_duration_for_full_weight)`**

**Parameters:**

*   `e_t`: The embedding of the new segment.
*   `μ_k,t-1`: The existing centroid for speaker `k`.
*   `segment_duration_seconds`: The duration of the audio segment in seconds.
*   `α_max`: A ceiling for the learning rate to prevent any single segment, no matter how long, from completely dominating the profile. A reasonable starting value is **0.25**.
*   `min_duration_for_full_weight`: The duration (in seconds) at which a segment is considered "full quality" and receives the maximum weight `α_max`. A good starting point is **4.0 seconds**.

**Example Behavior:**

*   A **0.5s** segment would have `Wt = min(0.25, 0.5 / 4.0) = 0.125`. It has a modest impact.
*   A **2.0s** segment would have `Wt = min(0.25, 2.0 / 4.0) = 0.25`. It has a strong impact.
*   A **5.0s** segment would have `Wt = min(0.25, 5.0 / 4.0) = 0.25`. It has the maximum allowed impact.

**Benefits:**

*   **Robustness:** Speaker profiles become highly resistant to corruption from fleeting, low-quality audio.
*   **Stability:** Centroids remain stable over long periods, only shifting in response to persistent, high-quality evidence of a voice change.
*   **Faster Convergence:** New speaker profiles stabilize more quickly if the initial enrollment segments are of good quality.

As before, the centroid `μ_k,t` **must be re-normalized to unit length** after every update to maintain the geometric validity of the Cosine Similarity metric.

---

## **2. Advanced Score Normalization: Self-Tuning Adaptive S-Norm**

The previous design correctly identified the need for score normalization using Adaptive S-Norm to make the decision threshold (`τ`) robust. However, it proposed using a generic, static cohort (e.g., from VoxCeleb). This is a critical limitation.

### **2.1. The Motivation: Adapting to the Local Acoustic Space**

A generic cohort does not represent the specific noise, channel conditions, or speaker demographics of the deployment environment. A system deployed in a call center will have a very different "background" acoustic space compared to one processing high-fidelity studio recordings. This mismatch leads to unreliable normalized scores and requires constant manual re-tuning of the decision threshold.

The system must learn the characteristics of its own environment.

### **2.2. The v2 Implementation: A Dynamic, Environment-Specific Cohort**

We will implement a mechanism for the system to automatically build and maintain a cohort of "imposter" embeddings sampled from its own operational data.

**The Logic Flow:**

1.  **Identify Confirmed-Imposter Segments:**
    *   During the decision logic, any incoming segment `e_t` that is a clear non-match (`S_best <= τ_new`) is flagged as a potential imposter.
    *   These embeddings are temporarily stored in a separate, dedicated "imposter candidates" collection in the database.

2.  **Background Cohort Refresh Process:**
    *   A periodic background job (e.g., running every hour or daily) is responsible for maintaining the production S-Norm cohort.
    *   This job randomly samples N embeddings (e.g., N=500) from the "imposter candidates" collection to form the new, active cohort.
    *   Using a rolling window for sampling (e.g., candidates from the last 7 days) ensures the cohort adapts to recent changes in the environment while maintaining stability.

3.  **Dynamic S-Norm Calculation:**
    *   The real-time identification process will load and use this dynamically generated cohort for all S-Norm calculations.
    *   When comparing an input `X` to a target `C_k`, it will normalize the raw score against the mean and standard deviation of the scores of `X` against this environment-specific cohort.

**Benefits:**

*   **Self-Tuning:** The system automatically adapts its decision threshold to the acoustic realities of its environment, dramatically reducing the need for manual tuning.
*   **Accuracy:** The normalized score becomes a much more reliable measure of speaker similarity, as it is contextualized by "what is typical" for that specific environment.
*   **Resilience:** The system becomes resilient to changes in the environment, such as the introduction of new background noise or a shift in microphone types used by the speaker population.

---

## **3. Implementation Status and Next Steps**

This section tracks the progress of implementing the v2 features described in this document.

### **3.1. Confidence-Weighted EMA (Section 1)**

*   **Status:** ✅ **Complete**
*   **Details:** The `titanet_identify_v2.py` script now successfully implements the Confidence-Weighted EMA for updating existing speaker profiles. The learning rate (`Wt`) is dynamically calculated based on the total duration of the speaker's segments in the current audio file. The Go adapter has been updated to pass all necessary parameters (`alpha_max`, `min_duration_for_full_weight`).

### **3.2. Self-Tuning Adaptive S-Norm (Section 2)**

*   **Status:** 🟡 **In Progress**

#### **Step 1: Imposter Candidate Collection**

*   **Status:** ✅ **Complete**
*   **Details:** The `titanet_identify_v2.py` script now identifies "confirmed imposter" segments. When a segment's best match score is below the `threshold_new`, its embedding is saved to a dedicated `imposter_candidates` collection in Qdrant.

#### **Step 2: Background Cohort Refresh Process**

*   **Status:** 🔴 **To Do**
*   **Next Steps:**
    1.  Create a new Python script, `titanet_cohort_manager.py`.
    2.  This script will contain a function to be run as a background job (e.g., via a cron job or a simple scheduler).
    3.  The function will:
        *   Connect to Qdrant.
        *   Retrieve a random sample of N embeddings (e.g., 500) from the `imposter_candidates` collection.
        *   Overwrite the contents of a `snorm_cohort` collection with the sampled embeddings.

#### **Step 3: Dynamic S-Norm Calculation**

*   **Status:** 🔴 **To Do**
*   **Next Steps:**
    1.  Modify the `identify_speakers` function in `titanet_identify_v2.py`.
    2.  Before the decision logic, add a step to fetch all vectors from the `snorm_cohort` collection.
    3.  For a given input `centroid`, calculate the cosine similarity against every vector in the cohort.
    4.  Compute the mean (`μ_cohort`) and standard deviation (`σ_cohort`) of these cohort scores.
    5.  Normalize the raw match score `S_best` using the formula: `S_norm = (S_best - μ_cohort) / σ_cohort`.
    6.  The final decision logic will then use `S_norm` against a new, normalized threshold (e.g., `τ_norm = 1.5`).
