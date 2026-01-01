package komodo

type CreateVariable struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
	IsSecret    bool   `json:"is_secret"`
}

type UpdateVariableIsSecret struct {
	Name     string `json:"name"`
	IsSecret bool   `json:"is_secret"`
}

type DeleteVariable struct {
	Name string `json:"name"`
}

type UpdateVariableValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UpdateVariableDescription struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Request struct {
	Type   string `json:"type"`
	Params any    `json:"params"`
}
