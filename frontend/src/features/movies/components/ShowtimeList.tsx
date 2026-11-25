import { Link } from 'react-router-dom';
import {
  formatDayHeading,
  formatDayKey,
  formatPriceCents,
  formatShowtimeTime,
} from '../../../shared/format';
import type { Showtime } from '../types';

export interface ShowtimeListProps {
  showtimes: Showtime[];
}

interface ShowtimeDayGroup {
  dayKey: string;
  label: string;
  items: Showtime[];
}

function groupShowtimesByDay(showtimes: Showtime[]): ShowtimeDayGroup[] {
  const groups: ShowtimeDayGroup[] = [];
  const byKey = new Map<string, ShowtimeDayGroup>();

  for (const showtime of showtimes) {
    const dayKey = formatDayKey(showtime.startsAt);
    let group = byKey.get(dayKey);
    if (!group) {
      group = {
        dayKey,
        label: formatDayHeading(showtime.startsAt),
        items: [],
      };
      byKey.set(dayKey, group);
      groups.push(group);
    }
    group.items.push(showtime);
  }

  return groups;
}

export function ShowtimeList({ showtimes }: ShowtimeListProps) {
  if (showtimes.length === 0) {
    return (
      <div className="empty-state">
        <p className="page-lead">No hay funciones próximas disponibles para esta película.</p>
      </div>
    );
  }

  const dayGroups = groupShowtimesByDay(showtimes);

  return (
    <div className="showtime-days">
      {dayGroups.map((group) => (
        <section key={group.dayKey} className="showtime-day-group">
          <h3 className="showtime-day-heading">{group.label}</h3>
          <ul className="showtime-list">
            {group.items.map((showtime) => (
              <li key={showtime.id}>
                <Link to={`/showtimes/${showtime.id}`} className="showtime-chip">
                  <span className="showtime-time">{formatShowtimeTime(showtime.startsAt)}</span>
                  <span className="showtime-auditorium">{showtime.auditorium}</span>
                  <span className="showtime-price">{formatPriceCents(showtime.priceCents)}</span>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
