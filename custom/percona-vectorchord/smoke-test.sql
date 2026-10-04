\set ON_ERROR_STOP on
CREATE EXTENSION vchord CASCADE;
CREATE EXTENSION earthdistance CASCADE;
CREATE EXTENSION "uuid-ossp";
CREATE EXTENSION unaccent;
CREATE EXTENSION pg_trgm;
CREATE TABLE vectorchord_test (id integer PRIMARY KEY, embedding vector(3));
INSERT INTO vectorchord_test SELECT i, ARRAY[i::float,1,1]::vector FROM generate_series(1,32) AS i;
CREATE INDEX vectorchord_test_index ON vectorchord_test USING vchordrq (embedding vector_cosine_ops) WITH (options=$$[build.internal]
lists = [1]
spherical_centroids = true$$);
SET enable_seqscan=off;
SET vchordrq.probes = '1';
SELECT id FROM vectorchord_test ORDER BY embedding <=> '[1,1,1]' LIMIT 1;
SELECT extname,extversion FROM pg_extension ORDER BY extname;
