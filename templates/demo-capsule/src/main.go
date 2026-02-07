package main

import "fmt"

// Contact represents a user's contact information.
type Contact struct {
	Name  string
	Email string
	Phone string
}

// FEATURE_GAP: ValidateEmail and ValidatePhone functions are not yet
// implemented. These are the tasks that bead fixtures will request.

func main() {
	c := Contact{
		Name:  "Alice",
		Email: "alice@example.com",
		Phone: "555-0100",
	}

	fmt.Printf("Contact: %s <%s> %s\n", c.Name, c.Email, c.Phone)
	fmt.Println("Note: input validation is not yet implemented")
}
