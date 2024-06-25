ALTER TABLE anki ADD column reviewed_int INT;

UPDATE anki SET reviewed_int = reviewed::INT;

ALTER TABLE anki DROP COLUMN reviewed;

ALTER TABLE anki RENAME COLUMN reviewed_int TO reviewed;

ALTER TABLE anki ADD column added_int INT;

UPDATE anki SET added_int = added_cards::INT;

ALTER TABLE anki DROP COLUMN added_cards;

ALTER TABLE anki RENAME COLUMN added_int TO added_cards;

ALTER TABLE anki ADD COLUMN new_time INTERVAL;

UPDATE anki SET new_time = time::interval;

ALTER TABLE anki DROP COLUMN time;

ALTER TABLE anki RENAME COLUMN new_time TO time;

ALTER TABLE books_history ADD column new_time INTERVAL;

UPDATE books_history SET new_time = time::INTERVAL;

ALTER TABLE books_history DROP COLUMN time;

ALTER TABLE books_history RENAME COLUMN new_time TO time;
