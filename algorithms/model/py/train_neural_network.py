import pandas as pd
import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim
from torch.utils.data import Dataset, DataLoader, TensorDataset
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler
from sklearn.metrics import mean_squared_error, mean_absolute_error, r2_score, accuracy_score
import pickle
import time
import os


class CompressionNet(nn.Module):
    """压缩参数预测神经网络"""
    def __init__(self, input_dim, hidden_dims=[256, 128, 64, 32], dropout=0.3):
        super(CompressionNet, self).__init__()
        
        layers = []
        prev_dim = input_dim
        
        # 构建隐藏层
        for i, hidden_dim in enumerate(hidden_dims):
            # 全连接层
            layers.append(nn.Linear(prev_dim, hidden_dim))
            # 批归一化
            layers.append(nn.BatchNorm1d(hidden_dim))
            # 激活函数
            layers.append(nn.ReLU())
            # Dropout (最后一层不加)
            if i < len(hidden_dims) - 1:
                layers.append(nn.Dropout(dropout))
            prev_dim = hidden_dim
        
        # 输出层 (4个参数)
        layers.append(nn.Linear(prev_dim, 4))
        
        self.network = nn.Sequential(*layers)
    
    def forward(self, x):
        return self.network(x)


def load_data():
    """
    加载训练数据
    特征: 61维 (基本统计12 + 数值特性6 + 差分统计16 + 时序3 + 游程4 + 位级5 + 分布3 + 自相关10 + 周期2)
    标签: 4个压缩参数，CSV列顺序为 [ranged(0), scale(1), del(2), algo(3)] - 与Go代码期望一致
    """
    print("正在加载数据...")
    features = pd.read_csv('../../../dataset/train_features.csv', header=None)
    labels = pd.read_csv('../../../dataset/train_labels.csv', header=None)
    
    print(f"特征数据形状: {features.shape}")
    print(f"标签数据形状: {labels.shape}")
    
    return features.values, labels.values


def prepare_data(X, y, test_size=0.2, random_state=42):
    """准备训练数据"""
    print("\n准备数据...")
    
    # 分割训练集和测试集
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=test_size, random_state=random_state
    )
    
    # 标准化特征
    scaler = StandardScaler()
    X_train_scaled = scaler.fit_transform(X_train)
    X_test_scaled = scaler.transform(X_test)
    
    print(f"训练集大小: {X_train.shape[0]}")
    print(f"测试集大小: {X_test.shape[0]}")
    print(f"特征维度: {X_train.shape[1]}")
    
    return X_train_scaled, X_test_scaled, y_train, y_test, scaler


def train_model(model, train_loader, test_loader, device, epochs=500, lr=0.001):
    """训练模型"""
    criterion = nn.MSELoss()
    optimizer = optim.Adam(model.parameters(), lr=lr, weight_decay=1e-5)
    scheduler = optim.lr_scheduler.ReduceLROnPlateau(
        optimizer, mode='min', factor=0.5, patience=5
    )
    
    best_loss = float('inf')
    best_epoch = 0
    
    train_losses = []
    test_losses = []
    
    print("\n开始训练...")
    print("="*80)
    print("⚠️  早停已禁用，将训练完整的 {} 个 epochs".format(epochs))
    
    for epoch in range(epochs):
        # 训练阶段
        model.train()
        train_loss = 0.0
        for batch_X, batch_y in train_loader:
            batch_X, batch_y = batch_X.to(device), batch_y.to(device)
            
            optimizer.zero_grad()
            outputs = model(batch_X)
            loss = criterion(outputs, batch_y)
            loss.backward()
            optimizer.step()
            
            train_loss += loss.item() * batch_X.size(0)
        
        train_loss /= len(train_loader.dataset)
        train_losses.append(train_loss)
        
        # 验证阶段
        model.eval()
        test_loss = 0.0
        with torch.no_grad():
            for batch_X, batch_y in test_loader:
                batch_X, batch_y = batch_X.to(device), batch_y.to(device)
                outputs = model(batch_X)
                loss = criterion(outputs, batch_y)
                test_loss += loss.item() * batch_X.size(0)
        
        test_loss /= len(test_loader.dataset)
        test_losses.append(test_loss)
        
        # 更新学习率
        old_lr = optimizer.param_groups[0]['lr']
        scheduler.step(test_loss)
        new_lr = optimizer.param_groups[0]['lr']
        
        # 如果学习率改变了，打印提示
        if old_lr != new_lr:
            print(f"Epoch {epoch+1}: 学习率从 {old_lr:.6f} 降低到 {new_lr:.6f}")
        
        # 打印进度
        if (epoch + 1) % 5 == 0 or epoch == 0:
            print(f"Epoch [{epoch+1:3d}/{epochs}] "
                  f"Train Loss: {train_loss:.6f} | "
                  f"Test Loss: {test_loss:.6f} | "
                  f"LR: {new_lr:.6f}")
        
        # 记录最佳损失（但不早停）
        if test_loss < best_loss:
            best_loss = test_loss
            best_epoch = epoch + 1
    
    print("="*80)
    print(f"训练完成! 最佳 Epoch: {best_epoch}, 最佳测试损失: {best_loss:.6f}")
    
    return train_losses, test_losses


def evaluate_model(model, X_test, y_test, device):
    """评估模型"""
    model.eval()
    
    X_test_tensor = torch.FloatTensor(X_test).to(device)
    
    with torch.no_grad():
        predictions = model(X_test_tensor).cpu().numpy()
    
    param_names = ['ranged', 'scale', 'del', 'algo']
    
    print("\n" + "="*80)
    print("模型评估结果")
    print("="*80)
    
    for i, name in enumerate(param_names):
        y_true = y_test[:, i]
        y_pred = predictions[:, i]
        y_pred_rounded = np.round(y_pred)
        
        mse = mean_squared_error(y_true, y_pred)
        mae = mean_absolute_error(y_true, y_pred)
        r2 = r2_score(y_true, y_pred)
        accuracy = accuracy_score(y_true, y_pred_rounded)
        
        print(f"\n参数 {i+1}: {name}")
        print(f"  MSE: {mse:.6f}")
        print(f"  MAE: {mae:.6f}")
        print(f"  R²:  {r2:.6f}")
        print(f"  准确率 (四舍五入): {accuracy:.4f} ({accuracy*100:.2f}%)")
        
        # 显示预测分布
        unique, counts = np.unique(y_pred_rounded.astype(int), return_counts=True)
        print(f"  预测分布: {dict(zip(unique, counts))}")


def save_model(model, scaler, save_dir='./'):
    """保存模型和标准化器"""
    print("\n" + "="*80)
    print("保存模型...")
    print("="*80)
    
    # 保存完整模型
    model_path = os.path.join(save_dir, 'neural_network_model.pth')
    torch.save(model.state_dict(), model_path)
    print(f"✅ 模型权重已保存: {model_path}")
    
    # 保存完整模型(包含结构)
    full_model_path = os.path.join(save_dir, 'neural_network_model_full.pth')
    torch.save(model, full_model_path)
    print(f"✅ 完整模型已保存: {full_model_path}")
    
    # 保存标准化器
    scaler_path = os.path.join(save_dir, 'neural_network_scaler.pkl')
    with open(scaler_path, 'wb') as f:
        pickle.dump(scaler, f)
    print(f"✅ 标准化器已保存: {scaler_path}")
    
    # 保存模型配置
    config = {
        'input_dim': model.network[0].in_features,
        'hidden_dims': [512, 256, 128, 64],
        'dropout': 0.3
    }
    config_path = os.path.join(save_dir, 'neural_network_config.pkl')
    with open(config_path, 'wb') as f:
        pickle.dump(config, f)
    print(f"✅ 模型配置已保存: {config_path}")


def main():
    print("="*80)
    print("神经网络压缩参数预测模型训练")
    print("="*80)
    
    # 设置随机种子
    torch.manual_seed(42)
    np.random.seed(42)
    
    # 检测设备
    device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    print(f"\n使用设备: {device}")
    if torch.cuda.is_available():
        print(f"GPU 名称: {torch.cuda.get_device_name(0)}")
    
    # 加载数据
    X, y = load_data()
    
    # 准备数据
    X_train, X_test, y_train, y_test, scaler = prepare_data(X, y)
    
    # 转换为 PyTorch 张量
    X_train_tensor = torch.FloatTensor(X_train)
    y_train_tensor = torch.FloatTensor(y_train)
    X_test_tensor = torch.FloatTensor(X_test)
    y_test_tensor = torch.FloatTensor(y_test)
    
    # 创建数据加载器
    batch_size = 256
    train_dataset = TensorDataset(X_train_tensor, y_train_tensor)
    test_dataset = TensorDataset(X_test_tensor, y_test_tensor)
    
    train_loader = DataLoader(train_dataset, batch_size=batch_size, shuffle=True)
    test_loader = DataLoader(test_dataset, batch_size=batch_size, shuffle=False)
    
    print(f"批次大小: {batch_size}")
    print(f"训练批次数: {len(train_loader)}")
    print(f"测试批次数: {len(test_loader)}")
    
    # 创建模型
    input_dim = X_train.shape[1]
    model = CompressionNet(
        input_dim=input_dim,
        hidden_dims=[256, 128, 64, 32],
        dropout=0.3
    ).to(device)
    
    print(f"\n模型结构:")
    print(model)
    
    # 计算参数量
    total_params = sum(p.numel() for p in model.parameters())
    trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)
    print(f"\n总参数量: {total_params:,}")
    print(f"可训练参数: {trainable_params:,}")
    
    # 训练模型
    start_time = time.time()
    train_losses, test_losses = train_model(
        model, train_loader, test_loader, device,
        epochs=500, lr=0.001
    )
    training_time = time.time() - start_time
    
    print(f"\n训练总时间: {training_time:.2f} 秒 ({training_time/60:.2f} 分钟)")
    
    # 评估模型
    evaluate_model(model, X_test, y_test, device)
    
    # 保存模型
    save_model(model, scaler)
    
    print("\n" + "="*80)
    print("训练完成!")
    print("="*80)


if __name__ == '__main__':
    main()
