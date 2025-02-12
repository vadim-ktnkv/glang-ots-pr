package hw04lrucache

type List interface {
	Len() int
	Front() *ListItem
	Back() *ListItem
	PushFront(v interface{}) *ListItem
	PushBack(v interface{}) *ListItem
	Remove(i *ListItem)
	MoveToFront(i *ListItem)
}

type ListItem struct {
	Value interface{}
	Next  *ListItem
	Prev  *ListItem
}

type list struct {
	length int
	head   *ListItem
	tail   *ListItem
}

var (
	nullNodeErr = "Item must not be nil"
	noNextErr   = "List item has no next node and doesn't belongs to tail, list corrupted"
	noPrevErr   = "List item has no prev node and doesn't belongs to head, list corrupted"
)

func (l *list) createNode(v interface{}) *ListItem {
	newNode := new(ListItem)
	newNode.Value = v
	return newNode
}

func (l *list) addFirstItem(newNode *ListItem) {
	l.head = newNode
	l.tail = newNode
	l.length++
}

func (l *list) Len() int {
	return l.length
}

func (l *list) Remove(i *ListItem) {
	if i == nil {
		panic(nullNodeErr)
	}

	if i.Next != nil {
		i.Next.Prev = i.Prev
	} else {
		if l.tail == i {
			l.tail = l.tail.Prev
		} else {
			panic(noNextErr)
		}
	}

	if i.Prev != nil {
		i.Prev.Next = i.Next
	} else {
		if l.head == i {
			l.head = l.head.Next
		} else {
			panic(noPrevErr)
		}
	}
	i.Value = nil
	i.Next = nil
	i.Prev = nil

	l.length--
}

func (l *list) MoveToFront(i *ListItem) {
	if i == nil {
		panic(nullNodeErr)
	}

	if l.head == i {
		return
	}

	if i.Prev == nil {
		panic(noPrevErr)
	}

	i.Prev.Next = i.Next

	if i.Next != nil {
		i.Next.Prev = i.Prev
	} else {
		if l.tail == i {
			l.tail = l.tail.Prev
		} else {
			panic(noNextErr)
		}
	}

	i.Prev = nil
	l.head.Prev = i
	i.Next = l.head
	l.head = i
}

func (l *list) Front() *ListItem {
	return l.head
}

func (l *list) Back() *ListItem {
	return l.tail
}

func (l *list) PushFront(v interface{}) *ListItem {
	newNode := l.createNode(v)

	if l.head == nil {
		l.addFirstItem(newNode)
		return newNode
	}
	l.head.Prev = newNode
	newNode.Next = l.head
	l.head = newNode
	l.length++
	return newNode
}

func (l *list) PushBack(v interface{}) *ListItem {
	newNode := l.createNode(v)

	if l.tail == nil {
		l.addFirstItem(newNode)
		return newNode
	}
	l.tail.Next = newNode
	newNode.Prev = l.tail
	l.tail = newNode
	l.length++
	return newNode
}

func NewList() List {
	return new(list)
}
