ALTER TABLE anki ADD column reviewed_int INT;

UPDATE anki SET reviewed_int = reviewed::INT;

ALTER TABLE anki DROP COLUMN reviewed;

ALTER TABLE anki RENAME COLUMN reviewed_int TO reviewed;

ALTER TABLE anki ADD column added_int INT;

UPDATE anki SET added_int = added_cards::INT;

ALTER TABLE anki DROP COLUMN added_cards;

ALTER TABLE anki RENAME COLUMN added_int TO added_cards;

