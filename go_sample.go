package main

import "fmt"

const JVAL = 5.87

type Customer struct {
	Name    string
	ID      int
	city    string
	Pincode int
}

func main() {
	var customer Customer
	fmt.Println("Customer struct:", customer)
	customer1 := Customer{
		Name:    "John Doe",
		ID:      101,
		city:    "New York",
		Pincode: 10001,
	}
	customer2 := Customer{"Jane Doe", 102, "Los Angeles", 90001}

	fmt.Println("Customer1 struct:", customer1)
	fmt.Println("Customer2 struct:", customer2)

	var arr [4]string
	var var1 int = 10
	var var2 float32 = 20.5 * JVAL
	var var3 bool = true

	fmt.Println("Integer:", var1)
	fmt.Println("Float:", var2)
	fmt.Println("Boolean:", var3)

	var integer_keymap map[int]string

	integer_keymap = map[int]string{
		1: "One",
		2: "Two",
		3: "Three",
		4: "Four",
		5: "Five"}

	if integer_keymap == nil {
		fmt.Println("Map is not initialized")
	} else {
		fmt.Println("Map is initialized")
	}

	for i := 0; i < var1; i++ {
		fmt.Println("Loop iteration:", i)
	}

	arr[0] = "Hello"
	arr[1] = "World"
	arr[2] = "!"
	arr[3] = "Go"

	fmt.Println(arr[0], arr[1], arr[2], arr[3])
	fmt.Println("Length of array:", len(arr))
	fmt.Println("Capacity of array:", cap(arr))

	fibonacci_number := [12]int{0, 1, 1, 2, 3, 5, 8, 13, 21, 34, 55, 89}
	fmt.Println("Fibonacci number at index 12:", fibonacci_number[11])

	fmt.Println("Fibonacci numbers: ", fibonacci_number)
}
