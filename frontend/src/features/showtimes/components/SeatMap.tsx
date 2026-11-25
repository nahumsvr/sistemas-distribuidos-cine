import type { Seat } from '../types';

export interface SeatMapProps {
  seats: Seat[];
  selectedSeatIds: ReadonlySet<number>;
  onToggleSeat: (seat: Seat) => void;
  disabled?: boolean;
}

interface SeatRowGroup {
  row: string;
  seats: Seat[];
}

function groupSeatsByRow(seats: Seat[]): SeatRowGroup[] {
  const rows: SeatRowGroup[] = [];
  const byRow = new Map<string, SeatRowGroup>();

  for (const seat of seats) {
    let group = byRow.get(seat.row);
    if (!group) {
      group = { row: seat.row, seats: [] };
      byRow.set(seat.row, group);
      rows.push(group);
    }
    group.seats.push(seat);
  }

  return rows;
}

export function SeatMap({
  seats,
  selectedSeatIds,
  onToggleSeat,
  disabled = false,
}: SeatMapProps) {
  const rows = groupSeatsByRow(seats);

  return (
    <div className="seat-map-wrapper">
      <div className="seat-screen" aria-hidden="true">
        PANTALLA
      </div>

      <ul className="seat-legend" aria-label="Leyenda de asientos">
        <li className="seat-legend-item">
          <span className="seat-swatch seat-swatch-available" aria-hidden="true" />
          <span>Disponible</span>
        </li>
        <li className="seat-legend-item">
          <span className="seat-swatch seat-swatch-selected" aria-hidden="true">
            ✓
          </span>
          <span>Seleccionado</span>
        </li>
        <li className="seat-legend-item">
          <span className="seat-swatch seat-swatch-occupied" aria-hidden="true">
            ×
          </span>
          <span>Ocupado</span>
        </li>
      </ul>

      <div className="seat-grid-scroll">
        <div className="seat-rows" role="group" aria-label="Mapa de asientos">
          {rows.map((rowGroup) => (
            <div key={rowGroup.row} className="seat-row">
              <span className="seat-row-label" aria-hidden="true">
                {rowGroup.row}
              </span>
              <div className="seat-row-seats">
                {rowGroup.seats.map((seat) => {
                  const isSelected = selectedSeatIds.has(seat.id);
                  const stateLabel = !seat.available
                    ? 'ocupado'
                    : isSelected
                      ? 'seleccionado'
                      : 'disponible';
                  const seatCode = `${seat.row}${seat.number}`;

                  return (
                    <button
                      key={seat.id}
                      type="button"
                      className={`seat-btn${!seat.available ? ' is-occupied' : isSelected ? ' is-selected' : ''}`}
                      disabled={!seat.available || disabled}
                      aria-pressed={seat.available ? isSelected : undefined}
                      aria-label={`Asiento ${seatCode}, ${stateLabel}`}
                      onClick={() => onToggleSeat(seat)}
                    >
                      <span className="seat-code">{seatCode}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
