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

