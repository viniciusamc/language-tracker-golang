CREATE TABLE public.anki (
	id serial4 NOT NULL,
	id_user uuid NOT NULL,
	reviewed varchar(8) NULL,
	added_cards varchar(8) NULL,
	"time" time NULL,
	target_language varchar(5) NOT NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT anki_pkey PRIMARY KEY (id),
	CONSTRAINT anki_id_user_foreign FOREIGN KEY (id_user) REFERENCES public.users(id) ON DELETE CASCADE
);
