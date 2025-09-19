package serializer

type Serializer[F, T any] interface {
	Serialize(from F) (T, error)
	Deserialize(to T) (F, error)
}
