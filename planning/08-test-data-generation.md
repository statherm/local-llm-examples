# Plan 08: Test Data Generation

**Status:** DRAFT
**Parent:** [00-project-overview.md](00-project-overview.md)
**Phase:** 3 (Exploration)

---

## 1. What This Demonstrates

A small local model generates synthetic but realistic test data — not random noise, but data that looks like it came from a real system. This is useful for development, testing, demos, and populating staging environments without using production data.

## 2. Why Small Models Should Shine Here

- **Pattern mimicry, not deep reasoning** — generating plausible names, addresses, transactions doesn't require genius
- **Schema-driven** — tell the model what fields to generate; it fills in plausible values
- **Volume** — you might need 10,000 records; $0 local generation is compelling
- **Privacy** — generating test data locally means no real data leaves your machine
- **Variety over perfection** — slightly imperfect data is fine for testing; diversity matters more

## 3. Example Scenarios

### 3a. User Profile Generation
**Input:** Schema definition + constraints ("US-based, ages 18-65, realistic names")
**Output:** JSON array of user profiles
**Scoring:** Schema compliance, distribution reasonableness, uniqueness

### 3b. Transaction Log Generation
**Input:** Schema + business rules ("transactions between $1-$10000, 5% should be flagged as suspicious")
**Output:** Timestamped transaction records
**Scoring:** Rule compliance, temporal consistency, statistical distribution

### 3c. API Response Mocking
**Input:** OpenAPI spec excerpt + example
**Output:** Realistic API response payloads
**Scoring:** Schema compliance, value plausibility

## 4. Open Questions

- How does small model test data quality compare to Faker/factory libraries? Where does LLM add value?
- Can small models maintain consistency across related records (e.g., user's city matches their zip code)?
- Is batch generation (100 records in one call) feasible with small context windows?
