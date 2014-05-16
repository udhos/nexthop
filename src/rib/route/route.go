package route

var (
	RouteAdd = make(chan Route)
	RouteDel = make(chan Route)
)
