package singgen

import "errors"

// Common errors that can occur during configuration generation
var (
	// ErrEmptySource is returned when no source URL or path is provided
	ErrEmptySource = errors.New("empty source URL or path")
	
	// ErrNoValidNodes is returned when no valid proxy nodes are found in the source
	ErrNoValidNodes = errors.New("no valid nodes found in source")
	
	// ErrUnsupportedFormat is returned when the output format is not supported
	ErrUnsupportedFormat = errors.New("unsupported output format")
	
	// ErrInvalidTemplate is returned when an invalid template version is specified
	ErrInvalidTemplate = errors.New("invalid template version")
	
	// ErrContextCanceled is returned when the operation is canceled via context
	ErrContextCanceled = errors.New("operation was canceled")
	
	// ErrInvalidPlatform is returned when an unsupported platform is specified
	ErrInvalidPlatform = errors.New("invalid platform")
	
	// ErrInvalidDNSServer is returned when an invalid DNS server address is provided
	ErrInvalidDNSServer = errors.New("invalid DNS server address")
	
	// ErrFetchFailed is returned when data fetching fails
	ErrFetchFailed = errors.New("failed to fetch data")
	
	// ErrParseFailed is returned when parsing fails
	ErrParseFailed = errors.New("failed to parse data")
	
	// ErrTransformFailed is returned when node transformation fails
	ErrTransformFailed = errors.New("failed to transform nodes")
	
	// ErrTemplateFailed is returned when template processing fails
	ErrTemplateFailed = errors.New("failed to process template")
	
	// ErrRenderFailed is returned when rendering the final configuration fails
	ErrRenderFailed = errors.New("failed to render configuration")
	
	// Multi-subscription related errors
	ErrNoSubscriptions = errors.New("no subscriptions configured")
	ErrEmptySubscriptionName = errors.New("subscription name cannot be empty")
	ErrEmptySubscriptionURL = errors.New("subscription URL cannot be empty")
	ErrDuplicateSubscriptionName = errors.New("duplicate subscription name")
	ErrConfigFileNotFound = errors.New("configuration file not found")
	ErrInvalidConfigFormat = errors.New("invalid configuration file format")
)