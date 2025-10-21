import pandas as pd
import numpy as np
import xgboost as xgb
import pickle
from sklearn.model_selection import train_test_split
from sklearn.metrics import mean_squared_error, mean_absolute_error, r2_score
import time

def load_data():
    """
    加载训练数据
    特征: 61维 (基本统计12 + 数值特性6 + 差分统计16 + 时序3 + 游程4 + 位级5 + 分布3 + 自相关10 + 周期2)
    标签: 4个压缩参数 (ranged, scale, del, algo)
    """
    print("正在加载数据...")
    features = pd.read_csv('../../../dataset/train_features.csv', header=None)
    labels = pd.read_csv('../../../dataset/train_labels.csv', header=None)
    
    print(f"特征数据形状: {features.shape}")
    print(f"标签数据形状: {labels.shape}")
    
    return features.values, labels.values

def train_xgboost_models():
    """使用 XGBoost 训练四个参数的模型"""
    
    # 加载数据
    X, y = load_data()
    
    # 分割训练集和测试集
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42
    )
    
    print(f"\n训练集大小: {X_train.shape[0]}")
    print(f"测试集大小: {X_test.shape[0]}")
    
    # 训练四个模型（对应四个参数）
    # CSV 列顺序是 [ranged(0), scale(1), del(2), algo(3)] - 与Go期望一致
    # 所以 models[0]=ranged, models[1]=scale, models[2]=del, models[3]=algo
    models = []
    param_names = ['ranged', 'scale', 'del', 'algo']  # 实际训练顺序
    
    for i in range(4):
        print(f"\n{'='*60}")
        print(f"训练参数 {i+1}/{4}: {param_names[i]}")
        print(f"{'='*60}")
        
        start_time = time.time()
        
        # 创建 XGBoost 回归器
        model = xgb.XGBRegressor(
            n_estimators=500,           # 树的数量
            learning_rate=0.05,         # 学习率
            max_depth=8,                # 树的最大深度
            min_child_weight=3,         # 子节点最小权重
            subsample=0.8,              # 行采样比例
            colsample_bytree=0.8,       # 列采样比例
            gamma=0.1,                  # 分裂所需最小损失减少
            reg_alpha=0.1,              # L1 正则化
            reg_lambda=1.0,             # L2 正则化
            random_state=42,
            n_jobs=-1,                  # 使用所有 CPU 核心
            verbosity=1,                # 打印训练信息
            early_stopping_rounds=50,   # 早停轮数
            eval_metric='rmse'          # 评估指标
        )
        
        # 训练模型
        model.fit(
            X_train, y_train[:, i],
            eval_set=[(X_test, y_test[:, i])],
            verbose=100  # 每100轮打印一次
        )
        
        # 预测
        y_pred_train = model.predict(X_train)
        y_pred_test = model.predict(X_test)
        
        # 计算评估指标
        train_mse = mean_squared_error(y_train[:, i], y_pred_train)
        test_mse = mean_squared_error(y_test[:, i], y_pred_test)
        train_mae = mean_absolute_error(y_train[:, i], y_pred_train)
        test_mae = mean_absolute_error(y_test[:, i], y_pred_test)
        train_r2 = r2_score(y_train[:, i], y_pred_train)
        test_r2 = r2_score(y_test[:, i], y_pred_test)
        
        # 计算准确率（四舍五入到整数后的准确率）
        train_acc = np.mean(np.round(y_pred_train) == y_train[:, i])
        test_acc = np.mean(np.round(y_pred_test) == y_test[:, i])
        
        elapsed_time = time.time() - start_time
        
        print(f"\n{param_names[i]} 训练完成! 耗时: {elapsed_time:.2f}秒")
        print(f"训练集 - MSE: {train_mse:.4f}, MAE: {train_mae:.4f}, R²: {train_r2:.4f}, 准确率: {train_acc:.4f}")
        print(f"测试集 - MSE: {test_mse:.4f}, MAE: {test_mae:.4f}, R²: {test_r2:.4f}, 准确率: {test_acc:.4f}")
        
        # 特征重要性
        feature_importance = model.feature_importances_
        top_features_idx = np.argsort(feature_importance)[-10:][::-1]
        print(f"\n前10个重要特征索引: {top_features_idx.tolist()}")
        print(f"重要度分数: {feature_importance[top_features_idx]}")
        
        models.append(model)
    
    return models, param_names

def save_models(models, param_names):
    """保存训练好的模型"""
    print(f"\n{'='*60}")
    print("保存模型...")
    print(f"{'='*60}")
    
    for i, (model, name) in enumerate(zip(models, param_names)):
        # 保存为 XGBoost 原生格式
        xgb_filename = f'xgboost_model_{name}.json'
        model.save_model(xgb_filename)
        print(f"✅ 模型 {i+1} ({name}) 已保存为: {xgb_filename}")
        
        # 同时保存为 pickle 格式（兼容性更好）
        pkl_filename = f'xgboost_model_{name}.pkl'
        with open(pkl_filename, 'wb') as f:
            pickle.dump(model, f)
        print(f"✅ 模型 {i+1} ({name}) 已保存为: {pkl_filename}")
    
    # 保存所有模型到一个文件
    all_models_filename = 'xgboost_models_all.pkl'
    with open(all_models_filename, 'wb') as f:
        pickle.dump(models, f)
    print(f"\n✅ 所有模型已保存到: {all_models_filename}")

if __name__ == '__main__':
    print("="*60)
    print("XGBoost 压缩参数预测模型训练")
    print("="*60)
    
    # 训练模型
    models, param_names = train_xgboost_models()
    
    # 保存模型
    save_models(models, param_names)
    
    print("\n" + "="*60)
    print("训练完成!")
    print("="*60)
