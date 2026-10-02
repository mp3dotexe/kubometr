package lasagnamaster

func PreparationTime(layers []string, timePerLayer int) int {
	if timePerLayer <= 0 {
		timePerLayer = 2
	}
	return len(layers) * timePerLayer
}

func Quantities(layer []string) (int, float64) {
	var noodles int
	var sauce float64

	for _, layers := range layer {
		if layers == "noodles" {
			noodles += 50
		} else if layers == "sauce" {
			sauce += 0.2
		}
	}

	return noodles, sauce
}

func AddSecretIngredient(friendList, myList []string) {
	if len(friendList) == 0 || len(myList) == 0 {
		return
	}

	secretIngredient := friendList[len(friendList)-1]

	myList[len(myList)-1] = secretIngredient
}

func ScaleRecipe(quantities []float64, portions int) []float64 {
	scaled := make([]float64, len(quantities))

	factor := float64(portions) / 2

	for i, qty := range quantities {
		scaled[i] = qty * factor
	}

	return scaled
}

// Your first steps could be to read through the tasks, and create
// these functions with their correct parameter lists and return types.
// The function body only needs to contain `panic("")`.
//
// This will make the tests compile, but they will fail.
// You can then implement the function logic one by one and see
// an increasing number of tests passing as you implement more
// functionality.
