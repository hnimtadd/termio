package datastruct

import "iter"

// An intrusive linked list is a data structure where the nodes are
// self-contained and manage their own links to other nodes.
type IntrusiveLinkedList[T comparable] struct {
	First *Node[T] // Pointer to the fist node
	Last  *Node[T] // Pointer to the next node
}

type Node[T any] struct {
	Next *Node[T] // Pointer to the next node in the list
	Prev *Node[T] // Pointer to the previous node in the list
	Data T        // The data contained in the node
}

func NewIntrusiveLinkedList[T comparable]() *IntrusiveLinkedList[T] {
	return &IntrusiveLinkedList[T]{
		First: nil,
		Last:  nil,
	}
}

// Insert a new node after an existing node.
func (l *IntrusiveLinkedList[T]) InsertAfter(node *Node[T], newNode *Node[T]) {
	newNode.Prev = node
	if nextNode := node.Next; nextNode != nil {
		// Intermediate node.
		newNode.Next = node.Next
		nextNode.Prev = newNode
	} else {
		// last element of the list.
		newNode.Next = nil
		l.Last = newNode
	}
	node.Next = newNode
}

// Insert a new node before an existing node.
func (l *IntrusiveLinkedList[T]) InsertBefore(node *Node[T], newNode *Node[T]) {
	newNode.Next = node
	if prevNode := node.Prev; prevNode != nil {
		// Intermediate node.
		newNode.Prev = node.Prev
		prevNode.Next = newNode
	} else {
		// First element of the list.
		newNode.Prev = nil
		l.First = newNode
	}
}

// Insert a new node at the end of the list.
func (l *IntrusiveLinkedList[T]) Append(newNode *Node[T]) {
	if lastNode := l.Last; lastNode != nil {
		// List is not empty, insert after the last node.
		l.InsertAfter(lastNode, newNode)
	} else {
		// List is empty, set first and last to the new node.
		l.Prepend(newNode)
	}
}

// Insert a new node at the beginning of the list.
func (l *IntrusiveLinkedList[T]) Prepend(newNode *Node[T]) {
	if firstNode := l.First; firstNode != nil {
		// Insert before the first node.
		l.InsertBefore(firstNode, newNode)
	} else {
		// Empty list
		l.First = newNode
		l.Last = newNode
		newNode.Prev = nil
		newNode.Next = nil
	}
}

// Remove a node from the list.
func (l *IntrusiveLinkedList[T]) Remove(node *Node[T]) {
	if prevNode := node.Prev; prevNode != nil {
		// Node is not the first element.
		prevNode.Next = node.Next
	} else {
		// Node is the first element.
		l.First = node.Next
	}

	if nextNode := node.Next; nextNode != nil {
		nextNode.Prev = node.Prev
	} else {
		// last element of the list.
		l.Last = node.Prev
	}
}

// Remove and return the last node in the list.
func (l *IntrusiveLinkedList[T]) Pop() *Node[T] {
	lastNode := l.Last
	if lastNode == nil {
		return nil
	}
	l.Remove(lastNode)
	return lastNode
}

// Remove and return the first node in the list.
func (l *IntrusiveLinkedList[T]) PopFirst() *Node[T] {
	firstNode := l.First
	if firstNode == nil {
		return nil
	}
	l.Remove(firstNode)
	return firstNode
}

// Search returns first node with the given value, or nil if not found.
func (l *IntrusiveLinkedList[T]) Search(val T) *Node[T] {
	for node := l.First; node != nil; node = node.Next {
		if node.Data == val {
			return node
		}
	}
	return nil
}

func (l *IntrusiveLinkedList[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {}
}
