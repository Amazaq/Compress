import pandas as pd
import numpy as np
from sklearn.ensemble import RandomForestClassifier, VotingClassifier
from sklearn.model_selection import train_test_split, GridSearchCV
from sklearn.metrics import accuracy_score, classification_report, f1_score
from sklearn.preprocessing import StandardScaler
from sklearn.neural_network import MLPClassifier
import xgboost as xgb
import lightgbm as lgb
import joblib
import os
import warnings
warnings.filterwarnings('ignore')

def load_data():
    """加载训练数据"""
    # 获取当前脚本的目录
    current_dir = os.path.dirname(os.path.abspath(__file__))
    # 构建数据集路径
    dataset_dir = os.path.join(current_dir, '..', '..', '..', 'dataset')
    
    features_path = os.path.join(dataset_dir, 'train_features.csv')
    labels_path = os.path.join(dataset_dir, 'train_labels.csv')
    
    # 加载特征数据
    features = pd.read_csv(features_path, header=None)
    # 加载标签数据
    labels = pd.read_csv(labels_path, header=None)
    
    print(f"Features shape: {features.shape}")
    print(f"Labels shape: {labels.shape}")
    
    return features.values, labels.values

def train_model():
    """训练模型"""
    print("Loading data...")
    X, y = load_data()
    
    # 分割训练集和测试集
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )
    
    print(f"Training set size: {X_train.shape[0]}")
    print(f"Test set size: {X_test.shape[0]}")
    
    # 数据标准化
    scaler = StandardScaler()
    X_train_scaled = scaler.fit_transform(X_train)
    X_test_scaled = scaler.transform(X_test)
    
    print("Training ensemble model with XGBoost, LightGBM, and Neural Network...")
    
    # 1. XGBoost分类器
    xgb_model = xgb.XGBClassifier(
        n_estimators=200,
        max_depth=6,
        learning_rate=0.1,
        subsample=0.8,
        colsample_bytree=0.8,
        random_state=42,
        n_jobs=-1,
        eval_metric='logloss'
    )
    
    # 2. LightGBM分类器
    lgb_model = lgb.LGBMClassifier(
        n_estimators=200,
        max_depth=6,
        learning_rate=0.1,
        subsample=0.8,
        colsample_bytree=0.8,
        random_state=42,
        n_jobs=-1,
        verbose=-1
    )
    
    # 3. 神经网络分类器
    nn_model = MLPClassifier(
        hidden_layer_sizes=(128, 64, 32),
        activation='relu',
        solver='adam',
        alpha=0.001,
        learning_rate='adaptive',
        max_iter=500,
        random_state=42
    )
    
    # 4. 随机森林分类器（改进版）
    rf_model = RandomForestClassifier(
        n_estimators=200,
        max_depth=10,
        min_samples_split=5,
        min_samples_leaf=2,
        random_state=42,
        n_jobs=-1
    )
    
    # 创建集成模型
    ensemble_model = VotingClassifier(
        estimators=[
            ('xgb', xgb_model),
            ('lgb', lgb_model),
            ('nn', nn_model),
            ('rf', rf_model)
        ],
        voting='soft'
    )
    
    # 训练集成模型
    print("Training XGBoost...")
    xgb_model.fit(X_train, y_train)
    
    print("Training LightGBM...")
    lgb_model.fit(X_train, y_train)
    
    print("Training Neural Network...")
    nn_model.fit(X_train_scaled, y_train)
    
    print("Training Random Forest...")
    rf_model.fit(X_train, y_train)
    
    print("Training Ensemble Model...")
    # 为集成模型准备数据（神经网络需要标准化数据）
    ensemble_model.fit(X_train_scaled, y_train)
    
    model = ensemble_model
    scaler_for_prediction = scaler
    
    # 预测和评估
    y_pred = model.predict(X_test_scaled)
    accuracy = accuracy_score(y_test, y_pred)
    f1 = f1_score(y_test, y_pred, average='weighted')
    
    print(f"\nEnsemble Model Performance:")
    print(f"Accuracy: {accuracy:.4f}")
    print(f"F1 Score: {f1:.4f}")
    print("\nClassification Report:")
    print(classification_report(y_test, y_pred))
    
    # 评估各个子模型的性能
    print("\nIndividual Model Performance:")
    
    # XGBoost
    xgb_pred = xgb_model.predict(X_test)
    xgb_acc = accuracy_score(y_test, xgb_pred)
    print(f"XGBoost Accuracy: {xgb_acc:.4f}")
    
    # LightGBM
    lgb_pred = lgb_model.predict(X_test)
    lgb_acc = accuracy_score(y_test, lgb_pred)
    print(f"LightGBM Accuracy: {lgb_acc:.4f}")
    
    # Neural Network
    nn_pred = nn_model.predict(X_test_scaled)
    nn_acc = accuracy_score(y_test, nn_pred)
    print(f"Neural Network Accuracy: {nn_acc:.4f}")
    
    # Random Forest
    rf_pred = rf_model.predict(X_test)
    rf_acc = accuracy_score(y_test, rf_pred)
    print(f"Random Forest Accuracy: {rf_acc:.4f}")
    
    # 保存模型和标准化器
    model_dir = os.path.dirname(os.path.abspath(__file__))
    model_path = os.path.join(model_dir, 'compression_model.pkl')
    scaler_path = os.path.join(model_dir, 'scaler.pkl')
    
    joblib.dump(model, model_path)
    joblib.dump(scaler_for_prediction, scaler_path)
    
    print(f"\nEnsemble model saved to: {model_path}")
    print(f"Scaler saved to: {scaler_path}")
    
    return model, scaler_for_prediction

if __name__ == "__main__":
    train_model()