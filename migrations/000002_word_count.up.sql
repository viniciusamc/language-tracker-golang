CREATE TABLE words (
	id SERIAL PRIMARY KEY,
	word VARCHAR UNIQUE 
);

CREATE INDEX idx_word ON words(word);

CREATE TABLE aux_words_amount (
	id SERIAL PRIMARY KEY,
	word INT NOT NULL REFERENCES words(id),
	amount INT NOT NULL,
	language varchar NOT NULL,
	id_user uuid REFERENCES users(id),
	UNIQUE(word, id_user)
);

CREATE UNIQUE INDEX unique_word_user ON aux_words_amount(word, id_user);
