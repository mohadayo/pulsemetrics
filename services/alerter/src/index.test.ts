import request from "supertest";
import { createApp, resetState } from "./index";

const app = createApp();

beforeEach(() => {
  resetState();
});

describe("GET /health", () => {
  it("returns ok status", async () => {
    const res = await request(app).get("/health");
    expect(res.status).toBe(200);
    expect(res.body.status).toBe("ok");
    expect(res.body.service).toBe("alerter");
  });
});

describe("POST /rules", () => {
  it("creates a valid rule", async () => {
    const res = await request(app).post("/rules").send({
      metricName: "cpu_usage",
      operator: "gt",
      threshold: 90,
      message: "CPU too high",
    });
    expect(res.status).toBe(201);
    expect(res.body.id).toBeDefined();
    expect(res.body.metricName).toBe("cpu_usage");
  });

  it("rejects missing metricName", async () => {
    const res = await request(app).post("/rules").send({
      operator: "gt",
      threshold: 90,
    });
    expect(res.status).toBe(422);
  });

  it("rejects invalid operator", async () => {
    const res = await request(app).post("/rules").send({
      metricName: "cpu",
      operator: "invalid",
      threshold: 90,
    });
    expect(res.status).toBe(422);
  });

  it("rejects non-numeric threshold", async () => {
    const res = await request(app).post("/rules").send({
      metricName: "cpu",
      operator: "gt",
      threshold: "high",
    });
    expect(res.status).toBe(422);
  });
});

describe("GET /rules", () => {
  it("returns empty array initially", async () => {
    const res = await request(app).get("/rules");
    expect(res.status).toBe(200);
    expect(res.body).toEqual([]);
  });

  it("returns created rules", async () => {
    await request(app).post("/rules").send({
      metricName: "cpu",
      operator: "gt",
      threshold: 80,
    });
    const res = await request(app).get("/rules");
    expect(res.body.length).toBe(1);
  });
});

describe("POST /evaluate", () => {
  it("triggers matching rule", async () => {
    await request(app).post("/rules").send({
      metricName: "cpu",
      operator: "gt",
      threshold: 80,
      message: "High CPU",
    });

    const res = await request(app).post("/evaluate").send({
      metricName: "cpu",
      value: 95,
    });
    expect(res.status).toBe(200);
    expect(res.body.triggeredCount).toBe(1);
    expect(res.body.alerts[0].message).toBe("High CPU");
  });

  it("does not trigger when below threshold", async () => {
    await request(app).post("/rules").send({
      metricName: "cpu",
      operator: "gt",
      threshold: 80,
    });

    const res = await request(app).post("/evaluate").send({
      metricName: "cpu",
      value: 50,
    });
    expect(res.body.triggeredCount).toBe(0);
  });

  it("rejects invalid input", async () => {
    const res = await request(app).post("/evaluate").send({
      metricName: "cpu",
    });
    expect(res.status).toBe(400);
  });
});

describe("GET /alerts", () => {
  it("returns triggered alerts", async () => {
    await request(app).post("/rules").send({
      metricName: "mem",
      operator: "lt",
      threshold: 100,
    });
    await request(app).post("/evaluate").send({
      metricName: "mem",
      value: 50,
    });

    const res = await request(app).get("/alerts");
    expect(res.status).toBe(200);
    expect(res.body.length).toBe(1);
  });
});
