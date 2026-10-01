package main

import (
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"
)

type Vitals struct {
	CPU, RAM      float64
	RAMUsedGB     float64
	RAMTotalGB    float64
	TempC         int
	NetIn, NetOut float64
	Load1, Load5  float64
	CPUHistory    []float64
}

type Dataset struct {
	Name   string
	UsedTB float64
	Tone   string
}

type Pool struct {
	Name     string
	Layout   string
	SizeTB   float64
	Health   string
	Datasets []Dataset
}

func (p Pool) UsedTB() float64 {
	var t float64
	for _, d := range p.Datasets {
		t += d.UsedTB
	}
	return t
}

type Disk struct {
	Bay         string
	Model       string
	Size        string
	TempC       int
	Status      string
	Hours       int
	Reallocated int
}

type Alert struct {
	Level string
	Text  string
	At    time.Time
}

type Service struct {
	Name     string
	Image    string
	Category string
	Port     int
	Status   string
	CPU      float64
	MemMB    int
	Started  time.Time
	settleAt time.Time
}

type Backup struct {
	ID       string
	Name     string
	Source   string
	Target   string
	Kind     string
	Schedule string
	SizeGB   float64
	LastRun  time.Time
	Status   string
	Note     string
	Progress int
	runStart time.Time
	runFor   time.Duration
}

type Coverage struct {
	Data                         string
	Primary, USB, Offsite, Cloud bool
}

func (c Coverage) Copies() int {
	n := 0
	for _, b := range []bool{c.Primary, c.USB, c.Offsite, c.Cloud} {
		if b {
			n++
		}
	}
	return n
}

func (c Coverage) Met() bool { return c.Copies() >= 3 && (c.Offsite || c.Cloud) }

type App struct {
	Key, Label string
}

var apps = []App{{"immich", "Photos"}, {"paperless", "Documents"}, {"jellyfin", "Media"}, {"pihole", "Network"}}

type User struct {
	ID       int
	Name     string
	Role     string
	Hue      int
	PhotosGB float64
	Docs     int
	Access   map[string]bool
}

type AccessChip struct {
	UserID int
	App    App
	On     bool
}

func (u User) Initial() string { return strings.ToUpper(u.Name[:1]) }

func (u User) Chips() []AccessChip {
	chips := make([]AccessChip, len(apps))
	for i, a := range apps {
		chips[i] = AccessChip{u.ID, a, u.Access[a.Key]}
	}
	return chips
}

type Document struct {
	ASN           int
	Title         string
	Correspondent string
	Type          string
	Tags          []string
	Added         time.Time
	Pages         int
}

type Album struct {
	Name  string
	Count int
	Hue   int
}

type Tile struct {
	Hue   int
	Label string
	Video bool
}

type Job struct {
	Name        string
	Done, Total int
	base        int
	rate        float64
}

func (j Job) Pct() int { return j.Done * 100 / j.Total }

type PhotoLib struct {
	Photos, Videos, People int
	SizeGB                 float64
	Albums                 []Album
	Recent                 []Tile
	Memory                 string
}

type Library struct {
	Name   string
	Count  string
	SizeTB float64
}

type Session struct {
	User, Title, Device string
	Progress            int
	Transcoding         bool
}

type MediaLib struct {
	Libraries []Library
	Sessions  []Session
	Recent    []Tile
}

type Bar struct{ Q, B int }

type Count struct {
	Name string
	N    int
}

type DNS struct {
	Enabled          bool
	Queries, Blocked int
	Clients, Gravity int
	Hourly           []Bar
	TopBlocked       []Count
	TopClients       []Count
}

func (d DNS) BlockedPct() float64 {
	if d.Queries == 0 {
		return 0
	}
	return float64(d.Blocked) / float64(d.Queries) * 100
}

type Snapshot struct {
	Host     string
	Uptime   time.Duration
	Vitals   Vitals
	Pools    []Pool
	Disks    []Disk
	Alerts   []Alert
	Services []Service
	Backups  []Backup
	Coverage []Coverage
	Users    []User
	Apps     []App
	Docs     []Document
	DocTotal int
	Inbox    int
	Tags     []string
	Photos   PhotoLib
	Jobs     []Job
	Media    MediaLib
	DNS      DNS
}

func (s Snapshot) ServicesUp() int {
	n := 0
	for _, svc := range s.Services {
		if svc.Status == "running" {
			n++
		}
	}
	return n
}

func (s Snapshot) Attention() int {
	n := 0
	for _, a := range s.Alerts {
		if a.Level == "warn" || a.Level == "bad" {
			n++
		}
	}
	return n
}

func (s Snapshot) BackupsOK() int {
	n := 0
	for _, b := range s.Backups {
		if b.Status == "ok" {
			n++
		}
	}
	return n
}

type Store struct {
	mu        sync.Mutex
	boot      time.Time
	jobsStart time.Time
	vitals    Vitals
	pools     []Pool
	disks     []Disk
	alerts    []Alert
	services  []*Service
	backups   []*Backup
	coverage  []Coverage
	users     []*User
	nextUser  int
	docs      []Document
	photos    PhotoLib
	jobs      []Job
	media     MediaLib
	dns       DNS
}

func (s *Store) Snapshot(q, tag string) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advance()

	snap := Snapshot{
		Host:     "hearth",
		Uptime:   time.Since(s.boot),
		Vitals:   s.vitals,
		Pools:    s.pools,
		Disks:    s.disks,
		Alerts:   s.alerts,
		Coverage: s.coverage,
		Apps:     apps,
		DocTotal: 1243,
		Photos:   s.photos,
		Jobs:     slices.Clone(s.jobs),
		Media:    s.media,
		DNS:      s.dns,
	}
	snap.Vitals.CPUHistory = slices.Clone(s.vitals.CPUHistory)
	snap.DNS.Hourly = slices.Clone(s.dns.Hourly)
	for _, svc := range s.services {
		snap.Services = append(snap.Services, *svc)
	}
	for _, b := range s.backups {
		snap.Backups = append(snap.Backups, *b)
	}
	for _, u := range s.users {
		cp := *u
		cp.Access = make(map[string]bool, len(u.Access))
		for k, v := range u.Access {
			cp.Access[k] = v
		}
		snap.Users = append(snap.Users, cp)
	}

	tags := map[string]bool{}
	q = strings.ToLower(strings.TrimSpace(q))
	for _, d := range s.docs {
		for _, t := range d.Tags {
			tags[t] = true
			if t == "inbox" {
				snap.Inbox++
			}
		}
		if tag != "" && !slices.Contains(d.Tags, tag) {
			continue
		}
		hay := strings.ToLower(d.Title + " " + d.Correspondent + " " + d.Type + " " + strings.Join(d.Tags, " "))
		if q != "" && !strings.Contains(hay, q) {
			continue
		}
		snap.Docs = append(snap.Docs, d)
	}
	for t := range tags {
		snap.Tags = append(snap.Tags, t)
	}
	slices.Sort(snap.Tags)
	return snap
}

// advance fakes a live system: every read nudges metrics and settles in-flight jobs.
func (s *Store) advance() {
	now := time.Now()
	v := &s.vitals
	v.CPU = clamp(v.CPU+rand.Float64()*14-7, 3, 94)
	v.CPUHistory = append(v.CPUHistory[1:], v.CPU)
	v.RAM = clamp(v.RAM+rand.Float64()*2-1, 38, 76)
	v.RAMUsedGB = v.RAM / 100 * v.RAMTotalGB
	v.TempC = 38 + int(v.CPU/6)
	v.NetIn = clamp(v.NetIn+rand.Float64()*20-10, 0.4, 112)
	v.NetOut = clamp(v.NetOut+rand.Float64()*6-3, 0.2, 40)
	v.Load1 = v.CPU / 100 * 4 * 1.1
	v.Load5 = (v.Load5*4 + v.Load1) / 5

	for _, svc := range s.services {
		if svc.Status == "restarting" && now.After(svc.settleAt) {
			svc.Status, svc.Started = "running", now
		}
		if svc.Status == "running" {
			svc.CPU = clamp(svc.CPU+rand.Float64()*2-1, 0.1, 60)
		} else {
			svc.CPU = 0
		}
	}

	for _, b := range s.backups {
		if b.Status != "running" {
			continue
		}
		el := now.Sub(b.runStart)
		if el >= b.runFor {
			b.Status, b.LastRun, b.Progress, b.Note = "ok", now, 100, ""
			continue
		}
		b.Progress = int(el * 100 / b.runFor)
	}

	el := now.Sub(s.jobsStart).Seconds()
	for i := range s.jobs {
		j := &s.jobs[i]
		j.Done = min(j.Total, j.base+int(el*j.rate))
	}

	if rand.IntN(2) == 0 {
		n := rand.IntN(18)
		s.dns.Queries += n
		s.dns.Hourly[23].Q += n
		if s.dns.Enabled {
			b := n * rand.IntN(4) / 10
			s.dns.Blocked += b
			s.dns.Hourly[23].B += b
		}
	}
}

func (s *Store) ServiceAction(name, action string) (Service, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, svc := range s.services {
		if svc.Name != name {
			continue
		}
		switch action {
		case "restart", "start":
			svc.Status, svc.settleAt = "restarting", time.Now().Add(2500*time.Millisecond)
		case "stop":
			svc.Status = "stopped"
		}
		return *svc, true
	}
	return Service{}, false
}

func (s *Store) Service(name string) (Service, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advance()
	for _, svc := range s.services {
		if svc.Name == name {
			return *svc, true
		}
	}
	return Service{}, false
}

func (s *Store) RunBackup(id string) (Backup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.backups {
		if b.ID != id {
			continue
		}
		if b.Status != "running" {
			b.Status, b.Progress, b.runStart = "running", 0, time.Now()
			b.runFor = time.Duration(4+rand.IntN(5)) * time.Second
		}
		return *b, true
	}
	return Backup{}, false
}

func (s *Store) Backup(id string) (Backup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advance()
	for _, b := range s.backups {
		if b.ID == id {
			return *b, true
		}
	}
	return Backup{}, false
}

func (s *Store) ToggleBlocking() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dns.Enabled = !s.dns.Enabled
}

var roleAccess = map[string][]string{
	"Admin": {"immich", "paperless", "jellyfin", "pihole"},
	"Adult": {"immich", "paperless", "jellyfin"},
	"Kid":   {"immich", "jellyfin"},
	"Guest": {"immich"},
}

func (s *Store) AddUser(name, role string) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := &User{ID: s.nextUser, Name: name, Role: role, Hue: rand.IntN(360), Access: map[string]bool{}}
	s.nextUser++
	for _, k := range roleAccess[role] {
		u.Access[k] = true
	}
	s.users = append(s.users, u)
	return *u
}

func (s *Store) DeleteUser(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = slices.DeleteFunc(s.users, func(u *User) bool { return u.ID == id })
}

func (s *Store) ToggleAccess(id int, app string) (AccessChip, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.ID != id {
			continue
		}
		for _, a := range apps {
			if a.Key == app {
				u.Access[app] = !u.Access[app]
				return AccessChip{u.ID, a, u.Access[app]}, true
			}
		}
	}
	return AccessChip{}, false
}

func clamp(v, lo, hi float64) float64 { return max(lo, min(hi, v)) }
