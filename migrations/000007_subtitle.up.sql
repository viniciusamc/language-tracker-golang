CREATE TABLE public.subtitles (
	id serial4 NOT NULL,
	id_media uuid NOT NULL,
	words varchar(128) NULL,
	CONSTRAINT subtitles_pkey PRIMARY KEY (id),
	CONSTRAINT subtitles_id_media_foreign FOREIGN KEY (id_media) REFERENCES public.medias(id) ON DELETE CASCADE
);
