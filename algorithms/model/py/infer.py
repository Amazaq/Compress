import argparse
import os
import sys
import joblib
import numpy as np
import pandas as pd


def main():
	parser = argparse.ArgumentParser(description="Run inference: read feature file and output predicted operation vector(s)")
	parser.add_argument("--model", default=os.path.join(os.path.dirname(__file__), "model.joblib"))
	parser.add_argument("--features", required=True, help="CSV file with features; each row is one sample")
	parser.add_argument("--out", default=None, help="Optional output CSV to write predictions")
	args = parser.parse_args()

	if not os.path.exists(args.model):
		raise FileNotFoundError(f"Model file not found: {args.model}. Train it first.")

	pipe = joblib.load(args.model)

	X = pd.read_csv(args.features, header=None).values.astype(np.float64)
	pred = pipe.predict(X)

	# Print to stdout and optionally write to file
	for row in pred:
		print("{},{},{}".format(int(row[0]), int(row[1]), int(row[2])))

	if args.out:
		pd.DataFrame(pred.astype(int)).to_csv(args.out, header=False, index=False)
		print(f"Saved predictions to {args.out}")


if __name__ == "__main__":
	main()
