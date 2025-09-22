import numpy as np
import joblib
import os
import sys
import json

class CompressionPredictor:
    def __init__(self):
        self.model = None
        self.scaler = None
        self.model_dir = os.path.dirname(os.path.abspath(__file__))
        self.model_path = os.path.join(self.model_dir, 'compression_model.pkl')
        self.scaler_path = os.path.join(self.model_dir, 'scaler.pkl')
        self.load_model()
    
    def load_model(self):
        """加载训练好的集成模型和标准化器"""
        try:
            if os.path.exists(self.model_path) and os.path.exists(self.scaler_path):
                self.model = joblib.load(self.model_path)
                self.scaler = joblib.load(self.scaler_path)
                print(f"Ensemble model loaded successfully from {self.model_path}", file=sys.stderr)
                print(f"Scaler loaded successfully from {self.scaler_path}", file=sys.stderr)
            else:
                missing_files = []
                if not os.path.exists(self.model_path):
                    missing_files.append(self.model_path)
                if not os.path.exists(self.scaler_path):
                    missing_files.append(self.scaler_path)
                raise FileNotFoundError(f"Model files not found: {missing_files}")
        except Exception as e:
            print(f"Error loading model: {e}", file=sys.stderr)
            sys.exit(1)
    
    def predict(self, features):
        """使用集成模型预测压缩算法组合
        
        Args:
            features: 输入特征数组
            
        Returns:
            预测的压缩算法向量
        """
        if self.model is None or self.scaler is None:
            raise RuntimeError("Model or scaler not loaded")
        
        # 确保输入是正确的形状
        if isinstance(features, list):
            features = np.array(features)
        
        if len(features.shape) == 1:
            features = features.reshape(1, -1)
        
        # 标准化特征（集成模型中的神经网络需要标准化数据）
        features_scaled = self.scaler.transform(features)
        
        # 进行预测
        prediction = self.model.predict(features_scaled)
        
        # 获取预测概率（可选，用于置信度评估）
        try:
            prediction_proba = self.model.predict_proba(features_scaled)
            confidence = np.max(prediction_proba, axis=1)
            print(f"Prediction confidence: {confidence[0]:.4f}", file=sys.stderr)
        except:
            pass
        
        return prediction[0].tolist() if len(prediction) == 1 else prediction.tolist()

def main():
    """主函数，用于命令行调用"""
    if len(sys.argv) != 2:
        print("Usage: python inference.py '<features_json>'", file=sys.stderr)
        print("Example: python inference.py '[1000.0, 0.509117, 21.028437, ...]'", file=sys.stderr)
        sys.exit(1)
    
    try:
        # 解析输入的特征
        features_json = sys.argv[1]
        features = json.loads(features_json)
        
        # 创建预测器并进行预测
        predictor = CompressionPredictor()
        result = predictor.predict(features)
        
        # 输出结果为JSON格式
        print(json.dumps(result))
        
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON input: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error during prediction: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()