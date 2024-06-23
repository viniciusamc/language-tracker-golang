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
