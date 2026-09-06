-- GridLine — griddb initialization script
-- Database: griddb
-- psql -h <aurora-writer-endpoint> -U <username> -d griddb -f init.sql

CREATE TABLE IF NOT EXISTS tracks (
    track_id  VARCHAR(16)  PRIMARY KEY,
    name      VARCHAR(100) NOT NULL,
    venue     VARCHAR(40)  NOT NULL,
    length_m  INTEGER      NOT NULL,
    status    VARCHAR(16)  NOT NULL
);

CREATE TABLE IF NOT EXISTS drivers (
    transponder_id VARCHAR(16)  PRIMARY KEY,
    display_name   VARCHAR(100) NOT NULL,
    team           VARCHAR(60)  NOT NULL,
    status         VARCHAR(16)  NOT NULL
);

CREATE TABLE IF NOT EXISTS heats (
    heat_id          UUID         PRIMARY KEY,
    track_id         VARCHAR(16)  NOT NULL REFERENCES tracks(track_id),
    label            VARCHAR(60)  NOT NULL,
    status           VARCHAR(16)  NOT NULL,
    duration_seconds INTEGER      NOT NULL,
    started_at       TIMESTAMPTZ  NULL,
    finished_at      TIMESTAMPTZ  NULL,
    created_at       TIMESTAMPTZ  NOT NULL
);

CREATE TABLE IF NOT EXISTS laps (
    lap_id          BIGSERIAL    PRIMARY KEY,
    heat_id         UUID         NOT NULL REFERENCES heats(heat_id),
    transponder_id  VARCHAR(16)  NOT NULL REFERENCES drivers(transponder_id),
    lap_number      INTEGER      NOT NULL,
    lap_time_ms     INTEGER      NOT NULL,
    crossed_at      TIMESTAMPTZ  NOT NULL,
    rules_version   VARCHAR(40)  NOT NULL,
    UNIQUE (heat_id, transponder_id, lap_number)
);

CREATE TABLE IF NOT EXISTS heat_standings (
    heat_id         UUID         NOT NULL REFERENCES heats(heat_id),
    transponder_id  VARCHAR(16)  NOT NULL REFERENCES drivers(transponder_id),
    position        INTEGER      NOT NULL,
    best_lap_ms     INTEGER      NOT NULL,
    last_lap_ms     INTEGER      NOT NULL,
    gap_ms          INTEGER      NOT NULL,
    updated_at      TIMESTAMPTZ  NOT NULL,
    PRIMARY KEY (heat_id, transponder_id)
);

CREATE INDEX IF NOT EXISTS idx_heats_status ON heats (status);
CREATE INDEX IF NOT EXISTS idx_laps_heat ON laps (heat_id, transponder_id);
CREATE INDEX IF NOT EXISTS idx_standings_heat ON heat_standings (heat_id, position);

INSERT INTO tracks (track_id, name, venue, length_m, status) VALUES
    ('track-01', 'Thunder Loop',       'indoor-a',  420, 'OPEN'),
    ('track-02', 'Neon Circuit',       'indoor-a',  380, 'OPEN'),
    ('track-03', 'Grid Hall Sprint',   'indoor-b',  310, 'OPEN'),
    ('track-04', 'Velo Dome',          'indoor-b',  450, 'OPEN'),
    ('track-05', 'Coast Kart Park',    'outdoor-c', 980, 'OPEN'),
    ('track-06', 'Ridge Outdoor',      'outdoor-d', 1120, 'OPEN')
ON CONFLICT (track_id) DO NOTHING;

INSERT INTO drivers (transponder_id, display_name, team, status) VALUES
    ('T-0001', 'Alex Mercer',    'Velocity Crew',   'ACTIVE'),
    ('T-0002', 'Blake Chen',     'Velocity Crew',   'ACTIVE'),
    ('T-0003', 'Casey Ortiz',    'Grid Racers',     'ACTIVE'),
    ('T-0004', 'Dana Kim',       'Grid Racers',     'ACTIVE'),
    ('T-0005', 'Elliot Nash',    'Thunder Kart',    'ACTIVE'),
    ('T-0006', 'Finn Oka',       'Thunder Kart',    'ACTIVE'),
    ('T-0007', 'Gina Park',      'Neon Pit',        'ACTIVE'),
    ('T-0008', 'Hugo Silva',     'Neon Pit',        'ACTIVE'),
    ('T-0009', 'Ivy Lund',       'Coastline',       'ACTIVE'),
    ('T-0010', 'Jonas Keller',   'Coastline',       'ACTIVE'),
    ('T-0011', 'Kiara Singh',    'Ridge Team',      'ACTIVE'),
    ('T-0012', 'Luis Ortega',    'Ridge Team',      'ACTIVE'),
    ('T-0013', 'Maya Lindqvist', 'Junior League',   'ACTIVE'),
    ('T-0014', 'Noah Berg',      'Junior League',   'ACTIVE'),
    ('T-0015', 'Olivia Chen',    'Velocity Crew',   'ACTIVE'),
    ('T-0016', 'Pavel Novak',    'Grid Racers',     'ACTIVE'),
    ('T-0017', 'Quinn Adler',    'Thunder Kart',    'ACTIVE'),
    ('T-0018', 'Rosa Alvarez',   'Neon Pit',        'ACTIVE'),
    ('T-0019', 'Samir Haddad',   'Coastline',       'ACTIVE'),
    ('T-0020', 'Tessa Okada',    'Ridge Team',      'ACTIVE'),
    ('T-0021', 'Uma Patel',      'Junior League',   'ACTIVE'),
    ('T-0022', 'Viktor Hale',    'Velocity Crew',   'ACTIVE'),
    ('T-0023', 'Wren Doyle',     'Grid Racers',     'ACTIVE'),
    ('T-0024', 'Xander Wu',      'Thunder Kart',    'ACTIVE'),
    ('T-0025', 'Yara Mensah',    'Neon Pit',        'ACTIVE'),
    ('T-0026', 'Zoe Martin',     'Coastline',       'ACTIVE'),
    ('T-0027', 'Aiden Fox',      'Ridge Team',      'ACTIVE'),
    ('T-0028', 'Bella Tran',     'Junior League',   'ACTIVE'),
    ('T-0029', 'Cole Ramsey',    'Velocity Crew',   'ACTIVE'),
    ('T-0030', 'Dina Kaur',      'Grid Racers',     'ACTIVE'),
    ('T-0031', 'Evan Brooks',    'Thunder Kart',    'ACTIVE'),
    ('T-0032', 'Freya Holt',     'Neon Pit',        'ACTIVE'),
    ('T-0033', 'Gabe Torres',    'Coastline',       'ACTIVE'),
    ('T-0034', 'Hana Ito',       'Ridge Team',      'ACTIVE'),
    ('T-0035', 'Ian Cole',       'Junior League',   'ACTIVE'),
    ('T-0036', 'Jade Myers',     'Velocity Crew',   'ACTIVE'),
    ('T-0037', 'Kai Ndiaye',     'Grid Racers',     'ACTIVE'),
    ('T-0038', 'Lena Vogt',      'Thunder Kart',    'ACTIVE'),
    ('T-0039', 'Milo Grant',     'Neon Pit',        'ACTIVE'),
    ('T-0040', 'Nina Rossi',     'Coastline',       'ACTIVE')
ON CONFLICT (transponder_id) DO NOTHING;
