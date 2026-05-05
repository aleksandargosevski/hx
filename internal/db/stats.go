package db

// CommandStat holds a command and its usage count.
type CommandStat struct {
	Command string
	Count   int
}

// DirectoryStat holds a directory and its usage count.
type DirectoryStat struct {
	Directory string
	Count     int
}

// HourStat holds an hour (0-23) and its command count.
type HourStat struct {
	Hour  int
	Count int
}

// FailStat holds a command, its failure count, and total count.
type FailStat struct {
	Command   string
	Failures  int
	Total     int
	FailRate  float64
}

// DurationStat holds a command and its average duration.
type DurationStat struct {
	Command    string
	AvgSeconds float64
	MaxSeconds int
	Count      int
}

// OverviewStats holds aggregate stats about the history database.
type OverviewStats struct {
	TotalCommands   int
	UniqueCommands  int
	TotalDirs       int
	DateRange       string
	AvgPerDay       float64
}

// StatsOverview returns aggregate stats about the history.
func (s *Store) StatsOverview() (*OverviewStats, error) {
	var stats OverviewStats

	err := s.db.QueryRow(
		`SELECT COUNT(*), COUNT(DISTINCT command), COUNT(DISTINCT directory)
		 FROM history WHERE deleted = 0`,
	).Scan(&stats.TotalCommands, &stats.UniqueCommands, &stats.TotalDirs)
	if err != nil {
		return nil, err
	}

	var minTs, maxTs int64
	err = s.db.QueryRow(
		`SELECT COALESCE(MIN(timestamp), 0), COALESCE(MAX(timestamp), 0)
		 FROM history WHERE deleted = 0`,
	).Scan(&minTs, &maxTs)
	if err != nil {
		return nil, err
	}

	if minTs > 0 && maxTs > 0 {
		days := float64(maxTs-minTs) / 86400.0
		if days < 1 {
			days = 1
		}
		stats.AvgPerDay = float64(stats.TotalCommands) / days
	}

	return &stats, nil
}

// TopCommands returns the most frequently used commands.
func (s *Store) TopCommands(limit int) ([]CommandStat, error) {
	rows, err := s.db.Query(
		`SELECT command, COUNT(*) as cnt
		 FROM history WHERE deleted = 0
		 GROUP BY command ORDER BY cnt DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CommandStat
	for rows.Next() {
		var cs CommandStat
		if err := rows.Scan(&cs.Command, &cs.Count); err != nil {
			return nil, err
		}
		results = append(results, cs)
	}
	return results, rows.Err()
}

// TopDirectories returns the most active directories.
func (s *Store) TopDirectories(limit int) ([]DirectoryStat, error) {
	rows, err := s.db.Query(
		`SELECT directory, COUNT(*) as cnt
		 FROM history WHERE deleted = 0 AND directory != ''
		 GROUP BY directory ORDER BY cnt DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DirectoryStat
	for rows.Next() {
		var ds DirectoryStat
		if err := rows.Scan(&ds.Directory, &ds.Count); err != nil {
			return nil, err
		}
		results = append(results, ds)
	}
	return results, rows.Err()
}

// CommandsByHour returns command counts grouped by hour of day.
func (s *Store) CommandsByHour() ([]HourStat, error) {
	rows, err := s.db.Query(
		`SELECT CAST(strftime('%H', timestamp, 'unixepoch', 'localtime') AS INTEGER) as hour,
		        COUNT(*) as cnt
		 FROM history WHERE deleted = 0 AND timestamp > 0
		 GROUP BY hour ORDER BY hour`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []HourStat
	for rows.Next() {
		var hs HourStat
		if err := rows.Scan(&hs.Hour, &hs.Count); err != nil {
			return nil, err
		}
		results = append(results, hs)
	}
	return results, rows.Err()
}

// MostFailing returns commands with the highest failure rates (min 3 executions).
func (s *Store) MostFailing(limit int) ([]FailStat, error) {
	rows, err := s.db.Query(
		`SELECT command,
		        SUM(CASE WHEN exit_code != 0 THEN 1 ELSE 0 END) as failures,
		        COUNT(*) as total
		 FROM history WHERE deleted = 0
		 GROUP BY command
		 HAVING total >= 3 AND failures > 0
		 ORDER BY CAST(failures AS REAL) / total DESC
		 LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FailStat
	for rows.Next() {
		var fs FailStat
		if err := rows.Scan(&fs.Command, &fs.Failures, &fs.Total); err != nil {
			return nil, err
		}
		fs.FailRate = float64(fs.Failures) / float64(fs.Total) * 100
		results = append(results, fs)
	}
	return results, rows.Err()
}

// SlowestCommands returns commands with the highest average duration.
func (s *Store) SlowestCommands(limit int) ([]DurationStat, error) {
	rows, err := s.db.Query(
		`SELECT command,
		        AVG(duration) as avg_dur,
		        MAX(duration) as max_dur,
		        COUNT(*) as cnt
		 FROM history WHERE deleted = 0 AND duration > 0
		 GROUP BY command
		 HAVING cnt >= 2
		 ORDER BY avg_dur DESC
		 LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DurationStat
	for rows.Next() {
		var ds DurationStat
		if err := rows.Scan(&ds.Command, &ds.AvgSeconds, &ds.MaxSeconds, &ds.Count); err != nil {
			return nil, err
		}
		results = append(results, ds)
	}
	return results, rows.Err()
}
