import express, { Request, Response } from "express";

const app = express();
app.use(express.json());

const PORT = parseInt(process.env.ALERTER_PORT || "8003", 10);
const LOG_LEVEL = process.env.LOG_LEVEL || "INFO";

function log(level: string, message: string): void {
  if (level === "DEBUG" && LOG_LEVEL !== "DEBUG") return;
  const ts = new Date().toISOString();
  console.log(`${ts} [${level}] alerter: ${message}`);
}

interface AlertRule {
  id: string;
  metricName: string;
  operator: "gt" | "lt" | "gte" | "lte" | "eq";
  threshold: number;
  message: string;
}

interface AlertEvent {
  ruleId: string;
  metricName: string;
  value: number;
  threshold: number;
  message: string;
  triggeredAt: string;
}

const rules: AlertRule[] = [];
const alerts: AlertEvent[] = [];
let ruleCounter = 0;

function evaluate(
  value: number,
  operator: string,
  threshold: number
): boolean {
  switch (operator) {
    case "gt":
      return value > threshold;
    case "lt":
      return value < threshold;
    case "gte":
      return value >= threshold;
    case "lte":
      return value <= threshold;
    case "eq":
      return value === threshold;
    default:
      return false;
  }
}

const VALID_OPERATORS = ["gt", "lt", "gte", "lte", "eq"];

app.get("/health", (_req: Request, res: Response) => {
  res.json({ status: "ok", service: "alerter" });
});

app.post("/rules", (req: Request, res: Response) => {
  const { metricName, operator, threshold, message } = req.body;

  if (!metricName || typeof metricName !== "string") {
    res.status(422).json({ error: "metricName is required and must be a string" });
    return;
  }
  if (!VALID_OPERATORS.includes(operator)) {
    res.status(422).json({ error: `operator must be one of: ${VALID_OPERATORS.join(", ")}` });
    return;
  }
  if (typeof threshold !== "number") {
    res.status(422).json({ error: "threshold must be a number" });
    return;
  }

  ruleCounter++;
  const rule: AlertRule = {
    id: `rule-${ruleCounter}`,
    metricName,
    operator,
    threshold,
    message: message || `${metricName} ${operator} ${threshold}`,
  };
  rules.push(rule);
  log("INFO", `Created rule ${rule.id}: ${rule.metricName} ${rule.operator} ${rule.threshold}`);
  res.status(201).json(rule);
});

app.get("/rules", (_req: Request, res: Response) => {
  res.json(rules);
});

app.post("/evaluate", (req: Request, res: Response) => {
  const { metricName, value } = req.body;

  if (!metricName || typeof value !== "number") {
    res.status(400).json({ error: "metricName and numeric value are required" });
    return;
  }

  const triggered: AlertEvent[] = [];
  for (const rule of rules) {
    if (rule.metricName === metricName && evaluate(value, rule.operator, rule.threshold)) {
      const event: AlertEvent = {
        ruleId: rule.id,
        metricName,
        value,
        threshold: rule.threshold,
        message: rule.message,
        triggeredAt: new Date().toISOString(),
      };
      alerts.push(event);
      triggered.push(event);
      log("INFO", `Alert triggered: ${rule.id} - ${rule.message}`);
    }
  }

  res.json({ evaluated: true, triggeredCount: triggered.length, alerts: triggered });
});

app.get("/alerts", (_req: Request, res: Response) => {
  res.json(alerts);
});

export function createApp() {
  return app;
}

export function resetState() {
  rules.length = 0;
  alerts.length = 0;
  ruleCounter = 0;
}

if (require.main === module) {
  app.listen(PORT, () => {
    log("INFO", `Starting alerter on port ${PORT}`);
  });
}
