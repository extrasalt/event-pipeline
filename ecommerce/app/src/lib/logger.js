const LEVELS = { debug: 0, info: 1, warn: 2, error: 3 };
const currentLevel = LEVELS.info;

function log(level, message, meta) {
  if (LEVELS[level] < currentLevel) return;
  const entry = { timestamp: new Date().toISOString(), level, message, meta };
  switch (level) {
    case "error": console.error(JSON.stringify(entry)); break;
    case "warn": console.warn(JSON.stringify(entry)); break;
    default: console.log(JSON.stringify(entry));
  }
}

export const logger = {
  debug: (msg, meta) => log("debug", msg, meta),
  info: (msg, meta) => log("info", msg, meta),
  warn: (msg, meta) => log("warn", msg, meta),
  error: (msg, meta) => log("error", msg, meta),
};
