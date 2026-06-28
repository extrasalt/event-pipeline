export function StarRating({ rate }) {
  return (
    <span className="text-yellow-500 text-sm" aria-label={`Rating: ${rate} out of 5`}>
      {"★".repeat(Math.round(rate))}{"☆".repeat(5 - Math.round(rate))}
    </span>
  );
}
