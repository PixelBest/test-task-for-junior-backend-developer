ALTER TABLE public.tasks
    ADD COLUMN periodicity_type TEXT NULL,
    ADD COLUMN periodicity INTEGER NULL,
    ADD COLUMN periodicity_dates DATE[] NULL,
    ADD COLUMN periodicity_closed BOOLEAN NOT NULL DEFAULT FALSE;