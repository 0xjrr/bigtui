package project

type Project struct {
	ID        string
	Location  string
	Resources []Resource
}

type Resource struct {
	Name string
	Kind string
}

func MockProjects() []Project {
	return []Project{
		{
			ID:       "sandbox-analytics",
			Location: "US",
			Resources: []Resource{
				{Name: "events", Kind: "dataset"},
				{Name: "events.raw_events", Kind: "table"},
				{Name: "events.daily_summary", Kind: "view"},
				{Name: "warehouse", Kind: "dataset"},
				{Name: "warehouse.orders", Kind: "table"},
			},
		},
		{
			ID:       "sandbox-reporting",
			Location: "EU",
			Resources: []Resource{
				{Name: "finance", Kind: "dataset"},
				{Name: "finance.monthly_revenue", Kind: "view"},
			},
		},
	}
}
