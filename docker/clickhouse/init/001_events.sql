CREATE TABLE IF NOT EXISTS default.events (
    id          String,
    type        String,
    timestamp   DateTime64(3),
    data        String,
    user_agent  String,
    timezone    String,
    location    String,
    session_id  String,
    churn_prob  Float64,
    param_count UInt32,
    inserted_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (type, timestamp);
