package main

import "fmt"

// Contact represents a person's contact information.
type Contact struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
	Street    string
	City      string
	State     string
	Zip       string
}

// FEATURE_GAP: ValidateEmail(email string) error
// Check for presence of @ and a domain with at least one dot.

// FEATURE_GAP: ValidatePhone(phone string) error
// Check for valid US phone pattern (digits, optional dashes/spaces/parens).

// FEATURE_GAP: FormatDisplayName(c Contact) string
// Combine FirstName and LastName into "Last, First" display format.

// FEATURE_GAP: FormatMailingAddress(c Contact) string
// Format Street, City, State, Zip into a multi-line mailing address.

func main() {
	c := Contact{
		FirstName: "Alice",
		LastName:  "Smith",
		Email:     "alice@example.com",
		Phone:     "555-0100",
		Street:    "123 Main St",
		City:      "Springfield",
		State:     "IL",
		Zip:       "62701",
	}

	fmt.Printf("Contact: %s %s <%s> %s\n", c.FirstName, c.LastName, c.Email, c.Phone)
	fmt.Printf("Address: %s, %s, %s %s\n", c.Street, c.City, c.State, c.Zip)
	fmt.Println("Note: validation and formatting functions are not yet implemented")
}
