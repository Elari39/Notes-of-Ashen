
UPDATE users
SET avatar_url = ''
WHERE TRIM(avatar_url) <> ''
  AND LOWER(TRIM(avatar_url)) NOT LIKE 'http://%'
  AND LOWER(TRIM(avatar_url)) NOT LIKE 'https://%';
