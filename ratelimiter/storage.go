package ratelimiter

type Storage interface {
	GetBucketFor(key string) (*LeakyBucket, error)
	SetBucketFor(key string, bucket LeakyBucket) error
}
