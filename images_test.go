package storage

import (
	"testing"
	"time"

	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func newTestImageStore(t *testing.T) rwImageStore {
	t.Helper()
	store, err := newImageStore(t.TempDir())
	require.Nil(t, err)
	return store
}

func addTestImage(t *testing.T, store rwImageStore, id string, names []string) {
	err := store.startWriting()
	require.NoError(t, err)
	defer store.stopWriting()

	_, err = store.Create(
		id, []string{}, "", "", time.Now(), digest.FromString(""),
	)

	require.Nil(t, err)
	require.Nil(t, store.updateNames(id, names, setNames))
}

func TestAddNameToHistorySuccess(t *testing.T) {
	// Given
	image := Image{}

	// When
	image.addNameToHistory("first")
	image.addNameToHistory("first")
	image.addNameToHistory("second")

	// Then
	require.Len(t, image.NamesHistory, 2)
}

func TestHistoryNames(t *testing.T) {
	// Given
	store := newTestImageStore(t)

	// When
	const firstImageID = "first"
	addTestImage(t, store, firstImageID, []string{"1", "2"})

	const secondImageID = "second"
	addTestImage(t, store, secondImageID, []string{"2", "3"})

	// Then
	firstImage, err := store.Get(firstImageID)
	require.Nil(t, err)
	require.Len(t, firstImage.Names, 1)
	require.Equal(t, firstImage.Names[0], "1")
	require.Len(t, firstImage.NamesHistory, 2)
	require.Equal(t, firstImage.NamesHistory[0], "2")
	require.Equal(t, firstImage.NamesHistory[1], "1")

	secondImage, err := store.Get(secondImageID)
	require.Nil(t, err)
	require.Len(t, secondImage.Names, 2)
	require.Equal(t, secondImage.Names[0], "2")
	require.Equal(t, secondImage.Names[1], "3")
	require.Len(t, secondImage.NamesHistory, 2)
	require.Equal(t, secondImage.NamesHistory[0], "3")
	require.Equal(t, secondImage.NamesHistory[1], "2")

	// And When
	require.NoError(t, store.startWriting())
	defer store.stopWriting()
	require.Nil(t, store.updateNames(firstImageID, []string{"1", "2", "3", "4"}, setNames))

	// Then
	firstImage, err = store.Get(firstImageID)
	require.Nil(t, err)
	require.Len(t, firstImage.NamesHistory, 4)
	require.Equal(t, firstImage.NamesHistory[0], "4")
	require.Equal(t, firstImage.NamesHistory[1], "3")
	require.Equal(t, firstImage.NamesHistory[2], "2")
	require.Equal(t, firstImage.NamesHistory[3], "1")

	secondImage, err = store.Get(secondImageID)
	require.Nil(t, err)
	require.Len(t, secondImage.Names, 0)
	require.Len(t, secondImage.NamesHistory, 2)
	require.Equal(t, secondImage.NamesHistory[0], "3")
	require.Equal(t, secondImage.NamesHistory[1], "2")

	// test independent add and remove operations
	require.Nil(t, store.updateNames(firstImageID, []string{"5"}, addNames))
	firstImage, err = store.Get(firstImageID)
	require.Nil(t, err)
	require.Equal(t, firstImage.NamesHistory, []string{"4", "3", "2", "1", "5"})

	// history should still contain old values
	require.Nil(t, store.updateNames(firstImageID, []string{"5"}, removeNames))
	firstImage, err = store.Get(firstImageID)
	require.Nil(t, err)
	require.Equal(t, firstImage.NamesHistory, []string{"4", "3", "2", "1", "5"})
}
