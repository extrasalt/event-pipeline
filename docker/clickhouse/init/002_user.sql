-- Create analytics user for remote API access
CREATE USER IF NOT EXISTS analytics IDENTIFIED WITH no_password;
GRANT ALL ON default.* TO analytics;

