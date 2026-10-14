import { describe, expect, it } from "vitest";

import { formatBytes, formatDuration } from "./format";

describe("formatBytes", () => {
  it("неположительные и NaN → «0 Б»", () => {
    expect(formatBytes(0)).toBe("0 Б");
    expect(formatBytes(-5)).toBe("0 Б");
    expect(formatBytes(Number.NaN)).toBe("0 Б");
  });

  it("байты без дробной части", () => {
    expect(formatBytes(512)).toBe("512 Б");
  });

  it("переводит в КБ/МБ/ГБ с одним знаком", () => {
    expect(formatBytes(1024)).toBe("1.0 КБ");
    expect(formatBytes(1024 * 1024)).toBe("1.0 МБ");
    expect(formatBytes(1024 ** 3)).toBe("1.0 ГБ");
  });
});

describe("formatDuration", () => {
  it("ноль/отрицательное/NaN → пустая строка", () => {
    expect(formatDuration(0)).toBe("");
    expect(formatDuration(-1)).toBe("");
    expect(formatDuration(Number.NaN)).toBe("");
  });

  it("минуты:секунды", () => {
    expect(formatDuration(187)).toBe("3:07");
  });

  it("часы:минуты:секунды", () => {
    expect(formatDuration(3725)).toBe("1:02:05");
  });
});
