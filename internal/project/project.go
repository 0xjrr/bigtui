package project

type Project struct {
	ID       string
	Location string
	Pinned   bool
}

var Defaults = []Project{
	{ID: "demo-analytics", Location: "US", Pinned: true},
	{ID: "warehouse-prod", Location: "EU", Pinned: true},
}
