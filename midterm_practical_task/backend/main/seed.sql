INSERT INTO wanted_criminals (name, nickname, reward, latitude, longitude, location_note, difficulty, armed, has_crew, escape_master)
SELECT * FROM (VALUES
  ('Pierrot Le Fou',   'Mad Pierrot',       70000000.00, NULL::float8, NULL::float8, 'Last seen near Mars orbital elevator', 10, true,  false, true),
  (NULL,               'Asteroid Ghost',     1200000.00, 34.05::float8, -118.25::float8, NULL,                                7, true,  true,  true),
  ('Vicious',          'The Dragon',         8500000.00, NULL::float8, NULL::float8, 'Ganymede red-light district',             9, true,  true,  false),
  ('Abdul Hakim',      'Teddy Bomber',       3000000.00, 25.20::float8, 55.27::float8, NULL,                                    6, true,  false, false),
  ('Dohn Dimitri',     'Space Smuggler',      450000.00, NULL::float8, NULL::float8, 'Titan cargo docks',                       4, false, true,  false),
  (NULL,               'Silent Whisper',     5200000.00, -33.87::float8, 151.21::float8, NULL,                                  8, true,  false, true),
  ('Mad Dog Jones',    'Rusty Pete',          150000.00, NULL::float8, NULL::float8, 'Unknown, somewhere in the Belt',          2, false, false, false)
) AS v(name, nickname, reward, latitude, longitude, location_note, difficulty, armed, has_crew, escape_master)
WHERE NOT EXISTS (SELECT 1 FROM wanted_criminals);

INSERT INTO hunt_logs (criminal_id, success, notes, created_at)
SELECT * FROM (VALUES
  (1, false, 'Target escaped through an asteroid field.',                NOW() - INTERVAL '12 days'),
  (1, false, 'Lost him in a hyperspace gate jam.',                       NOW() - INTERVAL '10 days'),
  (2, false, 'Ghost vanished before the ambush point.',                  NOW() - INTERVAL '9 days'),
  (2, true,  'Cornered on a derelict station, captured.',                NOW() - INTERVAL '6 days'),
  (3, false, 'Heavy resistance from his crew, retreated.',               NOW() - INTERVAL '8 days'),
  (4, true,  'Defused the bombs and caught him at the casino.',          NOW() - INTERVAL '7 days'),
  (4, false, 'False tip, wrong planet.',                                 NOW() - INTERVAL '14 days'),
  (5, true,  'Easy catch at the cargo docks.',                           NOW() - INTERVAL '5 days'),
  (6, false, 'Target spotted but fled in a stealth ship.',               NOW() - INTERVAL '4 days'),
  (6, false, 'Second attempt, tracker was destroyed.',                   NOW() - INTERVAL '3 days'),
  (7, true,  'Captured while drunk at a bar.',                           NOW() - INTERVAL '2 days'),
  (3, false, 'Ambush failed, ship damaged.',                             NOW() - INTERVAL '1 day')
) AS v(criminal_id, success, notes, created_at)
WHERE NOT EXISTS (SELECT 1 FROM hunt_logs);
