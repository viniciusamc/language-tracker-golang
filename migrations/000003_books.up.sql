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
