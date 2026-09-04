ALTER TABLE hosts ADD COLUMN IF NOT EXISTS username VARCHAR(128) NOT NULL DEFAULT '';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'credentials' AND column_name = 'username'
    ) THEN
        UPDATE hosts h
        SET username = c.username
        FROM credentials c
        WHERE h.credential_id = c.id
          AND COALESCE(h.username, '') = ''
          AND COALESCE(c.username, '') <> '';

        ALTER TABLE credentials DROP COLUMN IF EXISTS username;
    END IF;
END $$;
