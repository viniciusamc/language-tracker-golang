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
