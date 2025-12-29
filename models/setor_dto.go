package models

type FuncionarioResumo struct {
	ID            string `json:"id"`
	Nome          string `json:"nome"`
	TotalSemana   int    `json:"total_semana"`
	TotalMes      int    `json:"total_mes"`
	TotalExtraMes int    `json:"total_extra_mes"`
	Faltas        int    `json:"faltas"`
	Atestado      int    `json:"atestado"`
}

type SetorResumo struct {
	ID           string              `json:"id"`
	Setor        string              `json:"setor"`
	Funcionarios []FuncionarioResumo `json:"funcionarios"`
}
