package naming

import (
	"math/rand"
	"time"
)

var Cities = []string{
	"Abu Dhabi", "Accra", "Addis Ababa", "Ahmedabad", "Algiers", "Amman", "Amsterdam", "Anchorage", "Ankara", "Antananarivo",
	"Athens", "Atlanta", "Auckland", "Baku", "Bamako", "Bandung", "Bangkok", "Barcelona", "Beijing",
	"Beirut", "Belgrade", "Bengaluru", "Berlin", "Bogota", "Boston", "Bratislava", "Brisbane", "Brussels", "Bucharest",
	"Budapest", "Buenos Aires", "Cairo", "Calgary", "Cape Town", "Caracas", "Casablanca", "Chennai", "Chicago", "Copenhagen",
	"Dakar", "Dallas", "Delhi", "Denver", "Dhaka", "Doha", "Dubai", "Dublin", "Edinburgh",
	"Frankfurt", "Geneva", "Guangzhou", "Hanoi", "Harare", "Havana", "Helsinki", "Hong Kong", "Honolulu",
	"Houston", "Istanbul", "Jakarta", "Karachi", "Kathmandu", "Kigali",
	"Kingston", "Kinshasa", "Kuwait", "Lagos", "Lahore", "Lima", "Lisbon", "Ljubljana",
	"London", "Los Angeles", "Luanda", "Lusaka", "Madrid", "Manila", "Marrakesh", "Melbourne", "Mexico", "Miami",
	"Milan", "Minsk", "Montreal", "Mumbai", "Munich", "Nairobi", "Naples", "New Delhi", "New York",
	"Osaka", "Oslo", "Ottawa", "Paris", "Perth", "Phnom Penh", "Port Louis", "Prague", "Quito",
	"Reykjavik", "Riga", "Rio", "Riyadh", "Rome", "Rotterdam", "San Jose", "Santiago", "Sao Paulo",
	"Seattle", "Seoul", "Shanghai", "Singapore", "Sofia", "Stockholm", "Surabaya", "Sydney", "Taipei", "Tallinn",
	"Tashkent", "The Hague", "Tokyo", "Toronto", "Tunis", "Ulaanbaatar", "Vancouver",
	"Venice", "Vienna", "Vilnius", "Warsaw", "Washington", "Wellington", "Windhoek", "Yangon", "Yerevan", "Zagreb",
	"Alexandria", "Bergen", "Bilbao", "Birmingham", "Bordeaux", "Bristol", "Busan", "Cardiff", "Chongqing", "Cologne",
	"Curitiba", "Dalian", "Darwin", "Detroit", "Dresden", "Durban", "Florence", "Fukuoka", "Gaborone",
	"Gaziantep", "Gothenburg", "Guatemala", "Guayaquil", "Gwangju", "Halifax", "Hamburg", "Hangzhou", "Hiroshima", "Hobart",
	"Hyderabad", "Incheon", "Indore", "Izmir", "Jeddah", "Kampala", "Kanazawa", "Kano", "Kazan", "Kobe",
	"Kolkata", "Krakow", "La Paz", "Leeds", "Leipzig", "Lille", "Liverpool", "Lodz", "Marseille", "Medellin",
	"Mendoza", "Mombasa", "Monterrey", "Nagoya", "Nagasaki", "Nanjing", "Newcastle", "Nice", "Novosibirsk", "Odesa",
	"Oran", "Palermo", "Panama", "Patna", "Pattaya", "Peshawar", "Pittsburgh", "Podgorica", "Porto",
	"Poznan", "Quebec", "Rabat", "Recife", "Richmond", "Salvador", "Samarkand", "San Antonio", "San Diego",
	"Santa Cruz", "Sapporo", "Sarajevo", "Sendai", "Shenzhen", "Sheffield", "Shiraz", "Srinagar", "Stavanger",
	"Surat", "Suzhou", "Tbilisi", "Tijuana", "Toulouse", "Trabzon", "Turin", "Utrecht", "Valencia",
	"Valletta", "Verona", "Victoria", "Wuhan", "Xiamen", "Yokohama", "Zurich",
}

var source = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandomCity() string {
	return Cities[source.Intn(len(Cities))]
}
