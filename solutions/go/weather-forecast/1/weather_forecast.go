// Package weather weather package.
package weather 

// CurrentCondition accepts the current state.
var CurrentCondition string 
// CurrentLocation accepts the current location.
var CurrentLocation  string 

// Forecast Displays current weather conditions for current locations.
func Forecast(city, condition string) string {
	CurrentLocation, CurrentCondition = city, condition
	return CurrentLocation + " - current weather condition: " + CurrentCondition
}
