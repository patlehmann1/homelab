package main

import "time"

func NewStore() *Store {
	now := time.Now()
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	h, day := time.Hour, 24*time.Hour

	hist := make([]float64, 40)
	for i := range hist {
		hist[i] = 12 + float64((i*7)%19)
	}

	hourly := make([]Bar, 24)
	for i := range hourly {
		q := 900 + (i*137)%1400
		if i < 6 {
			q /= 3
		}
		hourly[i] = Bar{q, q / 5}
	}

	return &Store{
		boot:      ago(41*day + 6*h),
		jobsStart: now,
		nextUser:  5,
		vitals:    Vitals{CPU: 18, RAM: 52, RAMTotalGB: 32, NetIn: 12, NetOut: 3, Load5: 0.7, CPUHistory: hist},

		pools: []Pool{
			{Name: "tank", Layout: "RAID-Z1 · 4 × 8 TB", SizeTB: 21.8, Health: "ONLINE", Datasets: []Dataset{
				{"media", 6.42, "a"}, {"photos", 1.38, "b"}, {"backups", 0.91, "c"}, {"documents", 0.04, "d"}, {"other", 0.62, "e"},
			}},
			{Name: "flash", Layout: "mirror · 2 × 1 TB NVMe", SizeTB: 0.93, Health: "ONLINE", Datasets: []Dataset{
				{"app data", 0.22, "b"}, {"databases", 0.09, "c"}, {"cache", 0.05, "e"},
			}},
		},

		disks: []Disk{
			{"Bay 1", "WD Red Plus", "8 TB", 36, "ok", 9120, 0},
			{"Bay 2", "WD Red Plus", "8 TB", 37, "ok", 9118, 0},
			{"Bay 3", "IronWolf", "8 TB", 47, "warn", 14302, 8},
			{"Bay 4", "IronWolf", "8 TB", 39, "ok", 14290, 0},
			{"M.2 A", "SN850X", "1 TB", 44, "ok", 9100, 0},
			{"M.2 B", "SN850X", "1 TB", 43, "ok", 9100, 0},
		},

		alerts: []Alert{
			{"warn", "Bay 3 reports 8 reallocated sectors. Order a spare drive.", ago(2 * h)},
			{"bad", "Offsite backup to Mom & Dad's failed: host unreachable", ago(26 * h)},
			{"info", "Image update available for jellyfin", ago(9 * h)},
			{"ok", "Weekly scrub of tank finished with 0 errors", ago(3 * day)},
		},

		services: []*Service{
			{Name: "immich-server", Image: "ghcr.io/immich-app/immich-server:release", Category: "Photos", Port: 2283, Status: "running", CPU: 4.2, MemMB: 812, Started: ago(41 * day)},
			{Name: "immich-machine-learning", Image: "ghcr.io/immich-app/immich-machine-learning:release", Category: "Photos", Status: "running", CPU: 22.5, MemMB: 2410, Started: ago(41 * day)},
			{Name: "immich-postgres", Image: "ghcr.io/immich-app/postgres:16", Category: "Photos", Status: "running", CPU: 1.1, MemMB: 390, Started: ago(41 * day)},
			{Name: "paperless-ngx", Image: "ghcr.io/paperless-ngx/paperless-ngx:latest", Category: "Documents", Port: 8000, Status: "running", CPU: 0.8, MemMB: 640, Started: ago(12 * day)},
			{Name: "paperless-redis", Image: "redis:7", Category: "Documents", Status: "running", CPU: 0.2, MemMB: 34, Started: ago(12 * day)},
			{Name: "jellyfin", Image: "jellyfin/jellyfin:latest", Category: "Media", Port: 8096, Status: "running", CPU: 9.4, MemMB: 1120, Started: ago(6 * day)},
			{Name: "pihole", Image: "pihole/pihole:latest", Category: "Network", Port: 53, Status: "running", CPU: 0.6, MemMB: 88, Started: ago(41 * day)},
			{Name: "caddy", Image: "caddy:2", Category: "Network", Port: 443, Status: "running", CPU: 0.3, MemMB: 42, Started: ago(41 * day)},
			{Name: "backrest", Image: "garethgeorge/backrest:latest", Category: "Backups", Port: 9898, Status: "running", CPU: 0.4, MemMB: 96, Started: ago(41 * day)},
			{Name: "homelab-dash", Image: "homelab:dev", Category: "System", Port: 8080, Status: "running", CPU: 0.2, MemMB: 18, Started: ago(2 * h)},
		},

		backups: []*Backup{
			{ID: "snap", Name: "ZFS snapshots", Source: "tank/*", Target: "tank (local)", Kind: "local", Schedule: "hourly", SizeGB: 38, LastRun: ago(22 * time.Minute), Status: "ok"},
			{ID: "usb-photos", Name: "Photos → USB vault", Source: "tank/photos", Target: "usb-vault 14 TB", Kind: "local", Schedule: "nightly 02:00", SizeGB: 1380, LastRun: ago(8 * h), Status: "ok"},
			{ID: "b2-photos", Name: "Photos → Backblaze B2", Source: "tank/photos", Target: "b2://hearth-photos", Kind: "cloud", Schedule: "nightly 03:00", SizeGB: 1380, LastRun: ago(7 * h), Status: "ok"},
			{ID: "b2-docs", Name: "Documents → Backblaze B2", Source: "tank/documents", Target: "b2://hearth-docs", Kind: "cloud", Schedule: "nightly 03:30", SizeGB: 41, LastRun: ago(7 * h), Status: "ok"},
			{ID: "offsite", Name: "Everything important → Mom & Dad's", Source: "photos, documents, app data", Target: "tailscale: parents-pi", Kind: "offsite", Schedule: "weekly Sun 01:00", SizeGB: 1650, LastRun: ago(8 * day), Status: "failed", Note: "host unreachable"},
		},

		coverage: []Coverage{
			{"Photos & videos", true, true, true, true},
			{"Documents", true, false, true, true},
			{"App configs & DBs", true, true, true, false},
			{"Movies & shows", true, false, false, false},
		},

		users: []*User{
			{ID: 1, Name: "Patrick", Role: "Admin", Hue: 32, PhotosGB: 512, Docs: 640, Access: map[string]bool{"immich": true, "paperless": true, "jellyfin": true, "pihole": true}},
			{ID: 2, Name: "Alex", Role: "Adult", Hue: 190, PhotosGB: 488, Docs: 410, Access: map[string]bool{"immich": true, "paperless": true, "jellyfin": true}},
			{ID: 3, Name: "Maya", Role: "Kid", Hue: 320, PhotosGB: 61, Docs: 12, Access: map[string]bool{"immich": true, "jellyfin": true}},
			{ID: 4, Name: "Nana", Role: "Guest", Hue: 140, PhotosGB: 4, Access: map[string]bool{"immich": true}},
		},

		docs: []Document{
			{1243, "Electric bill: September", "Power Co.", "Bill", []string{"inbox", "utilities", "house"}, ago(5 * h), 2},
			{1242, "Pediatrician visit summary", "Neighborhood Pediatrics", "Medical record", []string{"inbox", "medical", "maya"}, ago(day), 3},
			{1241, "Dishwasher warranty card", "Bosch", "Warranty", []string{"inbox", "warranty", "house"}, ago(2 * day), 1},
			{1240, "Auto insurance renewal", "Progressive", "Policy", []string{"insurance", "auto"}, ago(4 * day), 14},
			{1239, "School supply receipt", "Target", "Receipt", []string{"receipts", "school"}, ago(6 * day), 1},
			{1238, "Report card, grade 3", "Elementary School", "Letter", []string{"school", "maya"}, ago(9 * day), 2},
			{1236, "Homeowners policy declarations", "Home Insurance Co.", "Policy", []string{"insurance", "house"}, ago(15 * day), 22},
			{1231, "Roof inspection report", "Coastal Roofing", "Report", []string{"house"}, ago(21 * day), 8},
			{1224, "W-2 2025", "Employer", "Tax form", []string{"taxes", "2025"}, ago(240 * day), 2},
			{1223, "1099-INT 2025", "Ally Bank", "Tax form", []string{"taxes", "2025"}, ago(242 * day), 1},
			{1219, "Federal return 2025", "IRS", "Tax return", []string{"taxes", "2025"}, ago(200 * day), 31},
			{1180, "Car title", "DMV", "Title", []string{"auto", "vital"}, ago(400 * day), 1},
			{1102, "Passport scan: Patrick", "U.S. Dept. of State", "ID", []string{"vital"}, ago(500 * day), 2},
			{1101, "Birth certificate: Maya", "Dept. of Health", "Certificate", []string{"vital", "maya"}, ago(500 * day), 1},
			{1090, "Mortgage closing disclosure", "Rocket Mortgage", "Contract", []string{"house", "vital"}, ago(700 * day), 5},
			{1077, "MacBook Pro receipt", "Apple", "Receipt", []string{"receipts", "warranty"}, ago(720 * day), 1},
			{1060, "Dental claim EOB", "Delta Dental", "Statement", []string{"medical", "insurance"}, ago(60 * day), 2},
		},

		photos: PhotoLib{
			Photos: 48213, Videos: 2194, People: 37, SizeGB: 1380,
			Memory: "On this day in 2021: 34 photos from the beach trip",
			Albums: []Album{
				{"Disney '25", 612, 210}, {"Christmas", 1408, 0}, {"Beach week", 389, 190}, {"Maya's art", 142, 300}, {"Launches", 97, 25}, {"Grandparents", 530, 120},
			},
			Recent: []Tile{
				{24, "Today", false}, {38, "Today", false}, {200, "Today", true}, {12, "Yesterday", false}, {190, "Yesterday", false}, {280, "Yesterday", false},
				{30, "Sep 28", false}, {160, "Sep 28", true}, {210, "Sep 27", false}, {350, "Sep 27", false}, {45, "Sep 26", false}, {120, "Sep 26", false},
				{260, "Sep 25", false}, {8, "Sep 25", false}, {175, "Sep 24", true}, {52, "Sep 24", false},
			},
		},

		jobs: []Job{
			{Name: "Thumbnails", Total: 412, base: 371, rate: 2},
			{Name: "Face detection", Total: 412, base: 220, rate: 1.4},
			{Name: "Smart search (CLIP)", Total: 412, base: 160, rate: 0.9},
			{Name: "Video transcode", Total: 18, base: 4, rate: 0.08},
		},

		media: MediaLib{
			Libraries: []Library{{"Movies", "412 films", 3.10}, {"Shows", "58 series · 1,904 episodes", 2.86}, {"Home videos", "316 clips", 0.40}, {"Music", "9,210 tracks", 0.21}},
			Sessions: []Session{
				{"Maya", "Bluey · S3E12 Housework", "iPad · Living room", 42, false},
				{"Patrick", "Apollo 13", "Apple TV · Den", 71, true},
			},
			Recent: []Tile{{18, "The Martian", false}, {205, "Planet Earth III", false}, {300, "Inside Out 2", false}, {42, "Hidden Figures", false}, {170, "Bluey · S3", false}, {230, "Cosmos", false}},
		},

		dns: DNS{
			Enabled: true, Queries: 48312, Blocked: 9871, Clients: 23, Gravity: 182544, Hourly: hourly,
			TopBlocked: []Count{{"doubleclick.net", 1822}, {"app-measurement.com", 1310}, {"graph.facebook.com", 944}, {"samsungads.com", 610}, {"device-metrics-us.amazon.com", 588}},
			TopClients: []Count{{"living-room-tv", 9120}, {"patrick-mbp", 7710}, {"alex-iphone", 6402}, {"maya-ipad", 5288}, {"hearth", 3011}},
		},
	}
}
