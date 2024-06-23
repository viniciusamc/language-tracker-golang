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
