package app

import (
	"math/rand"
	"time"
)

var worldCities = []string{
	"Abu Dhabi", "Accra", "Addis Ababa", "Ahmedabad", "Algiers", "Amman", "Amsterdam", "Anchorage", "Ankara", "Antananarivo",
	"Athens", "Atlanta", "Auckland", "Baku", "Bamako", "Bandung", "Bangkok", "Barcelona", "Beijing",
	"Beirut", "Belgrade", "Bengaluru", "Berlin", "Bogota", "Boston", "Bratislava", "Brisbane", "Brussels", "Bucharest",
	"Budapest", "Buenos Aires", "Cairo", "Calgary", "Cape Town", "Caracas", "Casablanca", "Chennai", "Chicago", "Copenhagen",
	"Dakar", "Dallas", "Dar es Salaam", "Delhi", "Denver", "Dhaka", "Doha", "Dubai", "Dublin", "Edinburgh",
	"Frankfurt", "Geneva", "Guangzhou", "Hanoi", "Harare", "Havana", "Helsinki", "Ho Chi Minh City", "Hong Kong", "Honolulu",
	"Houston", "Istanbul", "Jakarta", "Johannesburg", "Karachi", "Kathmandu", "Kigali",
	"Kingston", "Kinshasa", "Kuala Lumpur", "Kuwait City", "Lagos", "Lahore", "Lima", "Lisbon", "Ljubljana",
	"London", "Los Angeles", "Luanda", "Lusaka", "Madrid", "Manila", "Marrakesh", "Melbourne", "Mexico City", "Miami",
	"Milan", "Minsk", "Montreal", "Mumbai", "Munich", "Nairobi", "Naples", "New Delhi", "New York",
	"Osaka", "Oslo", "Ottawa", "Paris", "Perth", "Philadelphia", "Phnom Penh", "Port Louis", "Prague", "Quito",
	"Reykjavik", "Riga", "Rio de Janeiro", "Riyadh", "Rome", "Rotterdam", "San Francisco", "San Jose", "Santiago", "Sao Paulo",
	"Seattle", "Seoul", "Shanghai", "Singapore", "Sofia", "Stockholm", "Surabaya", "Sydney", "Taipei", "Tallinn",
	"Tashkent", "The Hague", "Tokyo", "Toronto", "Tunis", "Ulaanbaatar", "Vancouver",
	"Venice", "Vienna", "Vilnius", "Warsaw", "Washington", "Wellington", "Windhoek", "Yangon", "Yerevan", "Zagreb",
	"Alexandria", "Bergen", "Bilbao", "Birmingham", "Bordeaux", "Bristol", "Busan", "Cardiff", "Chongqing", "Cologne",
	"Curitiba", "Dalian", "Darwin", "Detroit", "Dresden", "Durban", "Florence", "Fukuoka", "Gaborone",
	"Gaziantep", "Gothenburg", "Guatemala City", "Guayaquil", "Gwangju", "Halifax", "Hamburg", "Hangzhou", "Hiroshima", "Hobart",
	"Hyderabad", "Incheon", "Indore", "Izmir", "Jeddah", "Kampala", "Kanazawa", "Kano", "Kazan", "Kobe",
	"Kolkata", "Krakow", "La Paz", "Leeds", "Leipzig", "Lille", "Liverpool", "Lodz", "Marseille", "Medellin",
	"Mendoza", "Mombasa", "Monterrey", "Nagoya", "Nagasaki", "Nanjing", "Newcastle", "Nice", "Novosibirsk", "Odesa",
	"Oran", "Palermo", "Panama City", "Patna", "Pattaya", "Peshawar", "Pittsburgh", "Podgorica", "Porto", "Porto Alegre",
	"Poznan", "Quebec City", "Rabat", "Recife", "Richmond", "Salvador", "Samarkand", "San Antonio", "San Diego", "San Salvador",
	"Santa Cruz", "Sapporo", "Sarajevo", "Sendai", "Shenzhen", "Sheffield", "Shiraz", "Srinagar", "Stavanger",
	"Surat", "Suzhou", "Tbilisi", "Thessaloniki", "Tijuana", "Toulouse", "Trabzon", "Turin", "Utrecht", "Valencia",
	"Valletta", "Verona", "Victoria", "Visakhapatnam", "Wuhan", "Xiamen", "Yokohama", "Zanzibar City", "Zurich",
}

var cityRandom = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomCity() string {
	return worldCities[cityRandom.Intn(len(worldCities))]
}
