-- public.knex_migrations definition

-- Drop table

-- DROP TABLE public.knex_migrations;

CREATE TABLE public.knex_migrations (
	id serial4 NOT NULL,
	"name" varchar(255) NULL,
	batch int4 NULL,
	migration_time timestamptz NULL,
	CONSTRAINT knex_migrations_pkey PRIMARY KEY (id)
);


-- public.knex_migrations_lock definition

-- Drop table

-- DROP TABLE public.knex_migrations_lock;

CREATE TABLE public.knex_migrations_lock (
	"index" serial4 NOT NULL,
	is_locked int4 NULL,
	CONSTRAINT knex_migrations_lock_pkey PRIMARY KEY (index)
);


-- public.users definition

-- Drop table

-- DROP TABLE public.users;

CREATE TABLE public.users (
	id uuid NOT NULL,
	username varchar(255) NOT NULL,
	email varchar(255) NOT NULL,
	"password" varchar(255) NOT NULL,
	configs jsonb NOT NULL,
	email_token uuid NULL,
	email_verified bool DEFAULT false NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT users_email_unique UNIQUE (email),
	CONSTRAINT users_pkey PRIMARY KEY (id),
	CONSTRAINT users_username_unique UNIQUE (username)
);


-- public.anki definition

-- Drop table

-- DROP TABLE public.anki;

CREATE TABLE public.anki (
	id serial4 NOT NULL,
	id_user uuid NOT NULL,
	reviewed varchar NULL,
	added_cards varchar NULL,
	"time" time NULL,
	target_language varchar(5) NOT NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT anki_pkey PRIMARY KEY (id),
	CONSTRAINT anki_id_user_foreign FOREIGN KEY (id_user) REFERENCES public.users(id) ON DELETE CASCADE
);


-- public.books definition

-- Drop table

-- DROP TABLE public.books;

CREATE TABLE public.books (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	id_user uuid NOT NULL,
	title varchar(256) NOT NULL,
	description text NULL,
	target_language varchar(5) NOT NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT books_pkey PRIMARY KEY (id),
	CONSTRAINT books_id_user_foreign FOREIGN KEY (id_user) REFERENCES public.users(id) ON DELETE CASCADE
);


-- public.books_history definition

-- Drop table

-- DROP TABLE public.books_history;

CREATE TABLE public.books_history (
	id serial4 NOT NULL,
	id_user uuid NOT NULL,
	id_book uuid NOT NULL,
	actual_page int4 NOT NULL,
	total_pages int4 NOT NULL,
	read_type varchar(64) NOT NULL,
	total_words int4 NOT NULL,
	"time" time NOT NULL,
	time_diff time NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT books_history_pkey PRIMARY KEY (id),
	CONSTRAINT books_history_id_book_foreign FOREIGN KEY (id_book) REFERENCES public.books(id) ON DELETE CASCADE,
	CONSTRAINT books_history_id_user_foreign FOREIGN KEY (id_user) REFERENCES public.users(id) ON DELETE CASCADE
);


-- public.medias definition

-- Drop table

-- DROP TABLE public.medias;

CREATE TABLE public.medias (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	id_user uuid NOT NULL,
	title varchar(128) NOT NULL,
	video_id varchar(128) NULL,
	episode varchar(255) NULL,
	"type" varchar(32) NOT NULL,
	watch_type varchar(32) NOT NULL,
	"time" time NULL,
	target_language varchar(8) NOT NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	total_words int4 DEFAULT 0 NULL,
	CONSTRAINT medias_pkey PRIMARY KEY (id),
	CONSTRAINT medias_id_user_foreign FOREIGN KEY (id_user) REFERENCES public.users(id) ON DELETE CASCADE
);


-- public."output" definition

-- Drop table

-- DROP TABLE public."output";

CREATE TABLE public."output" (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	id_user uuid NOT NULL,
	"type" varchar(128) NOT NULL,
	"time" time NOT NULL,
	summarize text NULL,
	target_language varchar(5) NOT NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT output_pkey PRIMARY KEY (id),
	CONSTRAINT output_id_user_foreign FOREIGN KEY (id_user) REFERENCES public.users(id) ON DELETE CASCADE
);


-- public.subtitles definition

-- Drop table

-- DROP TABLE public.subtitles;

CREATE TABLE public.subtitles (
	id serial4 NOT NULL,
	id_media uuid NOT NULL,
	words varchar(128) NULL,
	CONSTRAINT subtitles_pkey PRIMARY KEY (id),
	CONSTRAINT subtitles_id_media_foreign FOREIGN KEY (id_media) REFERENCES public.medias(id) ON DELETE CASCADE
);


-- public.vocabulary definition

-- Drop table

-- DROP TABLE public.vocabulary;

CREATE TABLE public.vocabulary (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	id_user uuid NOT NULL,
	vocabulary int4 NOT NULL,
	diff_last int4 NULL,
	url text NULL,
	target_language varchar(255) NOT NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT vocabulary_pkey PRIMARY KEY (id),
	CONSTRAINT vocabulary_id_user_foreign FOREIGN KEY (id_user) REFERENCES public.users(id) ON DELETE CASCADE
);
