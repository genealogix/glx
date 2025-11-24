package lib

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateRandomID generates a random 8-character hex ID using crypto/rand.
// Returns lowercase hex string like "a3f8d2c1".
// Uses 32 bits of randomness, which provides ~4.3 billion possible values.
// Collision probability is low even with thousands of entities.
func GenerateRandomID() (string, error) {
	// Generate 4 random bytes (32 bits)
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}

	// Convert to hex (8 characters)
	return hex.EncodeToString(bytes), nil
}

// GenerateEntityFilename generates a random filename for an entity.
// Format: {entity-type}-{random-id}.glx
// Example: person-a3f8d2c1.glx
//
// Entity types: person, event, relationship, place, source, repository, media, citation, assertion
func GenerateEntityFilename(entityType string) (string, error) {
	id, err := GenerateRandomID()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s.glx", entityType, id), nil
}

// GenerateUniqueFilename generates a unique filename that doesn't collide with existing filenames.
// Retries up to maxRetries times if collision is detected.
// Returns error if unable to generate unique filename after retries.
func GenerateUniqueFilename(entityType string, usedFilenames map[string]bool, maxRetries int) (string, error) {
	if maxRetries <= 0 {
		maxRetries = 10 // Default to 10 retries
	}

	for i := range maxRetries {
		filename, err := GenerateEntityFilename(entityType)
		if err != nil {
			return "", fmt.Errorf("failed to generate filename: %w", err)
		}

		// Check if already used
		if !usedFilenames[filename] {
			usedFilenames[filename] = true

			return filename, nil
		}

		// Collision detected, retry
		if i == maxRetries-1 {
			return "", fmt.Errorf("%w after %d retries", ErrUniqueFilenameFailed, maxRetries)
		}
	}

	return "", ErrUniqueFilenameFailed
}
