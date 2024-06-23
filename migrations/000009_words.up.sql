CREATE TABLE public.words (
	id serial4 NOT NULL,
	id_user uuid NOT NULL,
	id_media uuid NOT NULL,
	word varchar(256) NOT NULL,
	count int4 NOT NULL,
	target_language varchar(5) NOT NULL,
	CONSTRAINT words_pkey PRIMARY KEY (id),
	CONSTRAINT words_id_media_foreign FOREIGN KEY (id_media) REFERENCES public.medias(id),
	CONSTRAINT words_id_user_foreign FOREIGN KEY (id_user) REFERENCES public.users(id) ON DELETE CASCADE
);
