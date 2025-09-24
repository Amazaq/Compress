import argparse
import os
import joblib
import numpy as np
import pandas as pd
from sklearn.compose import ColumnTransformer
from sklearn.impute import SimpleImputer
from sklearn.metrics import classification_report
from sklearn.model_selection import train_test_split
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import StandardScaler
from sklearn.ensemble import RandomForestClassifier
from sklearn.multioutput import MultiOutputClassifier


def load_data(features_path: str, labels_path: str):
	X = pd.read_csv(features_path, header=None)
	Y = pd.read_csv(labels_path, header=None)
	# Ensure shapes align
	if X.shape[0] != Y.shape[0]:
		raise ValueError(f"Features rows {X.shape[0]} and labels rows {Y.shape[0]} mismatch")
	return X.values.astype(np.float64), Y.values.astype(np.int64)


def build_pipeline(random_state: int = 42) -> Pipeline:
	# Numeric pipeline with imputation and scaling
	numeric_transformer = Pipeline(steps=[
		("imputer", SimpleImputer(strategy="median")),
		("scaler", StandardScaler(with_mean=True, with_std=True)),
	])

	# Full pipeline (all columns numeric)
	preprocess = ColumnTransformer(
		transformers=[
			("num", numeric_transformer, slice(0, None)),
		]
	)

	base_clf = RandomForestClassifier(
		n_estimators=400,
		max_depth=None,
		min_samples_split=2,
		min_samples_leaf=1,
		max_features="sqrt",
		random_state=random_state,
		n_jobs=-1,
	)
	multi_clf = MultiOutputClassifier(base_clf, n_jobs=-1)

	pipe = Pipeline(steps=[
		("preprocess", preprocess),
		("clf", multi_clf),
	])
	return pipe


def main():
	parser = argparse.ArgumentParser(description="Train multi-output classifier for compression action selection")
	parser.add_argument("--features", default=os.path.join("..", "..", "..", "dataset", "train_features.csv"))
	parser.add_argument("--labels", default=os.path.join("..", "..", "..", "dataset", "train_labels.csv"))
	parser.add_argument("--out", default=os.path.join(os.path.dirname(__file__), "model.joblib"))
	parser.add_argument("--test_size", type=float, default=0.15)
	parser.add_argument("--random_state", type=int, default=42)
	args = parser.parse_args()

	X, Y = load_data(args.features, args.labels)

	X_train, X_test, Y_train, Y_test = train_test_split(
		X, Y, test_size=args.test_size, random_state=args.random_state, stratify=Y.values if isinstance(Y, pd.DataFrame) else Y
	)

	pipe = build_pipeline(random_state=args.random_state)
	pipe.fit(X_train, Y_train)

	Y_pred = pipe.predict(X_test)
	print("Evaluation on holdout set:")
	for i in range(Y.shape[1]):
		print(f"\nTask {i}:")
		print(classification_report(Y_test[:, i], Y_pred[:, i], digits=4))

	os.makedirs(os.path.dirname(args.out), exist_ok=True)
	joblib.dump(pipe, args.out)
	print(f"Saved model to {args.out}")


if __name__ == "__main__":
	main()
