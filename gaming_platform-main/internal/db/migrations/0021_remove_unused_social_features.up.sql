-- Posts, follows, and fraud reports were scaffolded from the original ERD
-- but never wired to any frontend feature (posts had a debug "Social Post"
-- panel, since removed; follows and fraud reports never had a UI at all).
-- Dropped to keep the schema matching what the app actually uses.
DROP TABLE posts;
DROP TABLE follows;
DROP TABLE fraud_reports;
