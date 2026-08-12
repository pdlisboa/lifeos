package jobs

import "time"

const maxBackoff = 30 * time.Minute

// nextRunAt aplica backoff exponencial: 2^attempts segundos, com teto de 30
// minutos. Função pura — testável sem banco.
func nextRunAt(now time.Time, attempts int16) time.Time {
	shift := attempts
	if shift > 30 { // previne overflow do shift; o teto de duração cuida do resto
		shift = 30
	}
	if shift < 0 {
		shift = 0
	}
	d := time.Duration(1<<uint(shift)) * time.Second
	if d > maxBackoff {
		d = maxBackoff
	}
	return now.Add(d)
}
