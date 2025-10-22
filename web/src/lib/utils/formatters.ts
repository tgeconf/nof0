import { format } from "date-fns";
import numeral from "numeral";

export const fmtUSD = (n?: number | null) =>
  n == null ? "--" : numeral(n).format("$0,0.00");

export const fmtPct = (n?: number | null) =>
  n == null ? "--" : numeral(n).format("0.00%");

export const fmtInt = (n?: number | null) =>
  n == null ? "--" : numeral(n).format("0,0");

export const fmtTs = (unixSeconds?: number | null) =>
  unixSeconds == null ? "--" : format(unixSeconds * 1000, "yyyy-MM-dd HH:mm:ss");

