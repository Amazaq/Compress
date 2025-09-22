import json
import numpy as np
from inference import CompressionPredictor
import pandas as pd
import os

def test_inference():
    """测试推理功能"""
    print("Testing ensemble model inference functionality...")
    
    # 检查模型文件是否存在
    model_dir = os.path.dirname(os.path.abspath(__file__))
    model_path = os.path.join(model_dir, 'compression_model.pkl')
    scaler_path = os.path.join(model_dir, 'scaler.pkl')
    
    if not os.path.exists(model_path) or not os.path.exists(scaler_path):
        print("Model or scaler file not found. Please run train.py first.")
        return False
    
    try:
        # 创建预测器
        predictor = CompressionPredictor()
        
        # 加载一些测试数据
        dataset_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', '..', 'dataset')
        features_path = os.path.join(dataset_dir, 'train_features.csv')
        
        if os.path.exists(features_path):
            # 读取前几行作为测试数据
            features_df = pd.read_csv(features_path, header=None)
            test_features = features_df.iloc[:5].values  # 取前5行
            
            print(f"Testing ensemble model with {len(test_features)} samples...")
            
            for i, features in enumerate(test_features):
                prediction = predictor.predict(features)
                print(f"Sample {i+1}: {prediction}")
            
            print("Ensemble model inference test completed successfully!")
            return True
        else:
            print(f"Test data file not found: {features_path}")
            return False
            
    except Exception as e:
        print(f"Error during inference test: {e}")
        return False

def test_json_interface():
    """测试JSON接口"""
    print("\nTesting JSON interface...")
    
    try:
        # 创建预测器
        predictor = CompressionPredictor()
        
        # 测试数据（来自train_features.csv的第一行）
        test_features = [
            1000.000000, 0.509117, 21.028437, 6.353100, 5.278507, 3.412313, 11.643878, 1.564762, 2.380408, 20.519320,
            3.010693, 4.198285, 7.208978, 322.000000, 0.322000, 0.000000, 0.000000, 1.000000, 0.001000, 0.000000,
            0.000000, -5.468616, 4.316300, -0.001350, 0.986565, 9.784916, 0.057057, 885.000000, 0.885000, -22.491619,
            2.833922, -6.359489, 3.753479, 25.325541, 0.000000, 962.000000, 0.962000, -0.004246, 3.753479, 410.000000,
            4.000000, 1.060445, 943.000000, 0.049841, 28.207000, 0.000000, 7.454428, 5.000000, 2.000000, 5.587758,
            13.783817, 2.901292, 0.959101, 0.897301, 0.833459, 0.769407, 0.712015, 0.663139, 0.621421, 0.583910,
            0.547904, 0.516522, 0.000000, 0.000000
        ]
        
        # 进行预测
        result = predictor.predict(test_features)
        
        # 转换为JSON并打印
        result_json = json.dumps(result)
        print(f"Prediction result (JSON): {result_json}")
        
        # 验证JSON可以被解析
        parsed_result = json.loads(result_json)
        print(f"Parsed result: {parsed_result}")
        
        print("JSON interface test completed successfully!")
        return True
        
    except Exception as e:
        print(f"Error during JSON interface test: {e}")
        return False

if __name__ == "__main__":
    print("Running tests for compression model...")
    print("=" * 50)
    
    # 运行测试
    test1_passed = test_inference()
    test2_passed = test_json_interface()
    
    print("\n" + "=" * 50)
    print("Test Results:")
    print(f"Inference test: {'PASSED' if test1_passed else 'FAILED'}")
    print(f"JSON interface test: {'PASSED' if test2_passed else 'FAILED'}")
    
    if test1_passed and test2_passed:
        print("\nAll tests passed! The model is ready for use.")
    else:
        print("\nSome tests failed. Please check the errors above.")