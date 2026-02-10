package database

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"nas-controller/internal/models"
)

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS apps (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		slug TEXT UNIQUE NOT NULL,
		description TEXT,
		icon TEXT,
		repo_url TEXT NOT NULL,
		branch TEXT NOT NULL,
		last_commit TEXT,
		last_pulled DATETIME,
		dockerfile_path TEXT DEFAULT './Dockerfile',
		build_context TEXT DEFAULT '.',
		build_args TEXT DEFAULT '{}',
		image_name TEXT,
		container_name TEXT,
		container_id TEXT,
		internal_port INTEGER DEFAULT 80,
		external_port INTEGER,
		restart_policy TEXT DEFAULT 'unless-stopped',
		env TEXT DEFAULT '{}',
		status TEXT DEFAULT 'stopped',
		last_build DATETIME,
		last_build_duration TEXT,
		last_build_success INTEGER DEFAULT 0,
		image_size INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_apps_slug ON apps(slug);
	CREATE INDEX IF NOT EXISTS idx_apps_status ON apps(status);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
	`

	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) CreateApp(app *models.App) error {
	buildArgsJSON, _ := json.Marshal(app.BuildArgs)
	envJSON, _ := json.Marshal(app.Env)

	_, err := db.conn.Exec(`
		INSERT INTO apps (
			id, name, slug, description, icon, repo_url, branch, last_commit, last_pulled,
			dockerfile_path, build_context, build_args, image_name, container_name, container_id,
			internal_port, external_port, restart_policy, env, status, last_build,
			last_build_duration, last_build_success, image_size, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		app.ID, app.Name, app.Slug, app.Description, app.Icon, app.RepoURL, app.Branch,
		app.LastCommit, app.LastPulled, app.DockerfilePath, app.BuildContext, string(buildArgsJSON),
		app.ImageName, app.ContainerName, app.ContainerID, app.InternalPort, app.ExternalPort,
		app.RestartPolicy, string(envJSON), app.Status, app.LastBuild, app.LastBuildDuration,
		app.LastBuildSuccess, app.ImageSize, app.CreatedAt, app.UpdatedAt,
	)
	return err
}

func (db *DB) UpdateApp(app *models.App) error {
	buildArgsJSON, _ := json.Marshal(app.BuildArgs)
	envJSON, _ := json.Marshal(app.Env)

	_, err := db.conn.Exec(`
		UPDATE apps SET
			name = ?, description = ?, icon = ?, repo_url = ?, branch = ?, last_commit = ?,
			last_pulled = ?, dockerfile_path = ?, build_context = ?, build_args = ?,
			image_name = ?, container_name = ?, container_id = ?, internal_port = ?,
			external_port = ?, restart_policy = ?, env = ?, status = ?, last_build = ?,
			last_build_duration = ?, last_build_success = ?, image_size = ?, updated_at = ?
		WHERE id = ?
	`,
		app.Name, app.Description, app.Icon, app.RepoURL, app.Branch, app.LastCommit,
		app.LastPulled, app.DockerfilePath, app.BuildContext, string(buildArgsJSON),
		app.ImageName, app.ContainerName, app.ContainerID, app.InternalPort,
		app.ExternalPort, app.RestartPolicy, string(envJSON), app.Status, app.LastBuild,
		app.LastBuildDuration, app.LastBuildSuccess, app.ImageSize, time.Now(), app.ID,
	)
	return err
}

func (db *DB) GetApp(id string) (*models.App, error) {
	row := db.conn.QueryRow(`SELECT * FROM apps WHERE id = ?`, id)
	return db.scanApp(row)
}

func (db *DB) GetAppBySlug(slug string) (*models.App, error) {
	row := db.conn.QueryRow(`SELECT * FROM apps WHERE slug = ?`, slug)
	return db.scanApp(row)
}

func (db *DB) GetAllApps() ([]*models.App, error) {
	rows, err := db.conn.Query(`SELECT * FROM apps ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*models.App
	for rows.Next() {
		app, err := db.scanAppRows(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (db *DB) DeleteApp(id string) error {
	_, err := db.conn.Exec(`DELETE FROM apps WHERE id = ?`, id)
	return err
}

func (db *DB) GetUsedPorts() ([]int, error) {
	rows, err := db.conn.Query(`SELECT external_port FROM apps WHERE external_port IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []int
	for rows.Next() {
		var port int
		if err := rows.Scan(&port); err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}
	return ports, nil
}

func (db *DB) scanApp(row *sql.Row) (*models.App, error) {
	app := &models.App{}
	var buildArgsJSON, envJSON string
	var lastPulled, lastBuild sql.NullTime
	var lastBuildSuccess int
	var containerID sql.NullString

	err := row.Scan(
		&app.ID, &app.Name, &app.Slug, &app.Description, &app.Icon, &app.RepoURL, &app.Branch,
		&app.LastCommit, &lastPulled, &app.DockerfilePath, &app.BuildContext, &buildArgsJSON,
		&app.ImageName, &app.ContainerName, &containerID, &app.InternalPort, &app.ExternalPort,
		&app.RestartPolicy, &envJSON, &app.Status, &lastBuild, &app.LastBuildDuration,
		&lastBuildSuccess, &app.ImageSize, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if lastPulled.Valid {
		app.LastPulled = &lastPulled.Time
	}
	if lastBuild.Valid {
		app.LastBuild = &lastBuild.Time
	}
	if containerID.Valid {
		app.ContainerID = containerID.String
	}
	app.LastBuildSuccess = lastBuildSuccess == 1
	json.Unmarshal([]byte(buildArgsJSON), &app.BuildArgs)
	json.Unmarshal([]byte(envJSON), &app.Env)

	if app.BuildArgs == nil {
		app.BuildArgs = make(map[string]string)
	}
	if app.Env == nil {
		app.Env = make(map[string]string)
	}

	return app, nil
}

func (db *DB) scanAppRows(rows *sql.Rows) (*models.App, error) {
	app := &models.App{}
	var buildArgsJSON, envJSON string
	var lastPulled, lastBuild sql.NullTime
	var lastBuildSuccess int
	var containerID sql.NullString

	err := rows.Scan(
		&app.ID, &app.Name, &app.Slug, &app.Description, &app.Icon, &app.RepoURL, &app.Branch,
		&app.LastCommit, &lastPulled, &app.DockerfilePath, &app.BuildContext, &buildArgsJSON,
		&app.ImageName, &app.ContainerName, &containerID, &app.InternalPort, &app.ExternalPort,
		&app.RestartPolicy, &envJSON, &app.Status, &lastBuild, &app.LastBuildDuration,
		&lastBuildSuccess, &app.ImageSize, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if lastPulled.Valid {
		app.LastPulled = &lastPulled.Time
	}
	if lastBuild.Valid {
		app.LastBuild = &lastBuild.Time
	}
	if containerID.Valid {
		app.ContainerID = containerID.String
	}
	app.LastBuildSuccess = lastBuildSuccess == 1
	json.Unmarshal([]byte(buildArgsJSON), &app.BuildArgs)
	json.Unmarshal([]byte(envJSON), &app.Env)

	if app.BuildArgs == nil {
		app.BuildArgs = make(map[string]string)
	}
	if app.Env == nil {
		app.Env = make(map[string]string)
	}

	return app, nil
}

// Session management
func (db *DB) CreateSession(token string, expiresAt time.Time) error {
	_, err := db.conn.Exec(`INSERT INTO sessions (token, expires_at) VALUES (?, ?)`, token, expiresAt)
	return err
}

func (db *DB) ValidateSession(token string) bool {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ? AND expires_at > ?`, token, time.Now()).Scan(&count)
	return err == nil && count > 0
}

func (db *DB) DeleteSession(token string) error {
	_, err := db.conn.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (db *DB) CleanupExpiredSessions() error {
	_, err := db.conn.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	return err
}
